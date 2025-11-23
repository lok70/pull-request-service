package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pull-request-service/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PRRepo реализует репозиторий pull request'ов на базе PostgreSQL.
type PRRepo struct {
	db *Postgres
}

// NewPRRepo создаёт новый экземпляр PRRepo c переданным подключением к PostgreSQL.
func NewPRRepo(db *Postgres) *PRRepo {
	return &PRRepo{db: db}
}

// CreatePRWithReviewers создаёт pull request и привязывает к нему указанных ревьюверов
// в рамках одной транзакции. При конфликте по идентификатору PR вернёт ErrPRExists.
func (r *PRRepo) CreatePRWithReviewers(
	ctx context.Context,
	pr model.PullRequest,
	reviewerIDs []string,
) (model.PullRequest, error) {

	// 1. Получаем исполнителя (Транзакцию или Пул)
	q := r.db.GetQueryExecutor(ctx)

	// 2. Выполняем запросы через q, а не через r.db.Pool
	row := q.QueryRow(ctx, `
INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
VALUES ($1, $2, $3, $4)
RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
`, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, string(pr.Status))

	var created model.PullRequest
	var status string
	var createdAt time.Time
	var mergedAt *time.Time

	if err := row.Scan(&created.PullRequestID, &created.PullRequestName, &created.AuthorID, &status, &createdAt, &mergedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.PullRequest{}, ErrPRExists
		}
		return model.PullRequest{}, fmt.Errorf("insert pr: %w", err)
	}
	created.Status = model.PullRequestStatus(status)
	created.CreatedAt = &createdAt
	created.MergedAt = mergedAt
	created.AssignedReviewers = make([]string, 0)

	if len(reviewerIDs) > 0 {
		batch := &pgx.Batch{}
		for _, rid := range reviewerIDs {
			batch.Queue(`
INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id)
VALUES ($1, $2)
`, created.PullRequestID, rid)
		}
		// Используем q для отправки батча
		br := q.SendBatch(ctx, batch)
		if err := br.Close(); err != nil {
			return model.PullRequest{}, fmt.Errorf("insert reviewers: %w", err)
		}
	}

	// Для чтения ревьюверов тоже передаем q, чтобы видеть изменения внутри текущей транзакции
	reviewers, err := r.listReviewersWithExecutor(ctx, q, created.PullRequestID)
	if err != nil {
		return model.PullRequest{}, err
	}
	created.AssignedReviewers = reviewers

	return created, nil
}

// GetPR возвращает pull request по идентификатору вместе со списком его ревьюверов.
// Если PR не найден, возвращает ErrPRNotFound.
func (r *PRRepo) GetPR(ctx context.Context, prID string) (model.PullRequest, error) {
	row := r.db.Pool.QueryRow(ctx, `
SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
FROM pull_requests
WHERE pull_request_id = $1
`, prID)

	var pr model.PullRequest
	var status string
	var createdAt time.Time
	var mergedAt *time.Time

	if err := row.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &status, &createdAt, &mergedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.PullRequest{}, ErrPRNotFound
		}
		return model.PullRequest{}, fmt.Errorf("get pr: %w", err)
	}

	pr.Status = model.PullRequestStatus(status)
	pr.CreatedAt = &createdAt
	pr.MergedAt = mergedAt

	reviewers, err := r.listReviewers(ctx, r.db.Pool, pr.PullRequestID)
	if err != nil {
		return model.PullRequest{}, err
	}
	pr.AssignedReviewers = reviewers

	return pr, nil
}

// MarkMerged помечает pull request как MERGED и устанавливает время мержа (если оно ещё не установлено).
// Если PR не найден, возвращает ErrPRNotFound.
func (r *PRRepo) MarkMerged(ctx context.Context, prID string, mergedAt time.Time) (model.PullRequest, error) {
	row := r.db.Pool.QueryRow(ctx, `
UPDATE pull_requests
SET status = 'MERGED',
    merged_at = COALESCE(merged_at, $2)
WHERE pull_request_id = $1
RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
`, prID, mergedAt)

	var pr model.PullRequest
	var status string
	var createdAt time.Time
	var mergedAtOut *time.Time

	if err := row.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &status, &createdAt, &mergedAtOut); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.PullRequest{}, ErrPRNotFound
		}
		return model.PullRequest{}, fmt.Errorf("update pr: %w", err)
	}

	pr.Status = model.PullRequestStatus(status)
	pr.CreatedAt = &createdAt
	pr.MergedAt = mergedAtOut

	reviewers, err := r.listReviewers(ctx, r.db.Pool, pr.PullRequestID)
	if err != nil {
		return model.PullRequest{}, err
	}
	pr.AssignedReviewers = reviewers

	return pr, nil
}

// ReassignReviewer заменяет ревьювера oldUserID на newUserID в указанном PR.
// Если строка не найдена (PR или ревьювер не привязан), возвращает ErrPRNotFound.
func (r *PRRepo) ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) (model.PullRequest, error) {
	cmdTag, err := r.db.Pool.Exec(ctx, `
UPDATE pull_request_reviewers
SET reviewer_id = $3
WHERE pull_request_id = $1 AND reviewer_id = $2
`, prID, oldUserID, newUserID)
	if err != nil {
		return model.PullRequest{}, fmt.Errorf("update reviewer: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return model.PullRequest{}, ErrPRNotFound
	}

	return r.GetPR(ctx, prID)
}

// ListAssignedToUser возвращает список укороченных описаний PR,
// в которых указанный пользователь назначен ревьювером.
func (r *PRRepo) ListAssignedToUser(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	rows, err := r.db.Pool.Query(ctx, `
SELECT pr.pull_request_id,
       pr.pull_request_name,
       pr.author_id,
       pr.status
FROM pull_requests pr
JOIN pull_request_reviewers r
  ON pr.pull_request_id = r.pull_request_id
WHERE r.reviewer_id = $1
ORDER BY pr.created_at DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("query pull requests: %w", err)
	}
	defer rows.Close()

	res := make([]model.PullRequestShort, 0)
	for rows.Next() {
		var pr model.PullRequestShort
		var status string
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &status); err != nil {
			return nil, fmt.Errorf("scan pr: %w", err)
		}
		pr.Status = model.PullRequestStatus(status)
		res = append(res, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return res, nil
}

// listReviewers возвращает список идентификаторов ревьюверов для заданного PR.
func (r *PRRepo) listReviewers(ctx context.Context, q pgxQuerier, prID string) ([]string, error) {
	rows, err := q.Query(ctx, `
SELECT reviewer_id
FROM pull_request_reviewers
WHERE pull_request_id = $1
ORDER BY reviewer_id
`, prID)
	if err != nil {
		return nil, fmt.Errorf("query reviewers: %w", err)
	}
	defer rows.Close()

	res := make([]string, 0)
	for rows.Next() {
		var rid string
		if err := rows.Scan(&rid); err != nil {
			return nil, fmt.Errorf("scan reviewer: %w", err)
		}
		res = append(res, rid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return res, nil
}

// pgxQuerier описывает минимальный интерфейс для выполнения запросов,
// который реализуют как пул соединений, так и транзакция pgx.
type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// listReviewersWithExecutor — вспомогательный метод (нужно обновить старый listReviewers)
func (r *PRRepo) listReviewersWithExecutor(ctx context.Context, q DBTX, prID string) ([]string, error) {
	rows, err := q.Query(ctx, `
SELECT reviewer_id
FROM pull_request_reviewers
WHERE pull_request_id = $1
ORDER BY reviewer_id
`, prID)
	if err != nil {
		return nil, fmt.Errorf("query reviewers: %w", err)
	}
	defer rows.Close()

	res := make([]string, 0)
	for rows.Next() {
		var rid string
		if err := rows.Scan(&rid); err != nil {
			return nil, fmt.Errorf("scan reviewer: %w", err)
		}
		res = append(res, rid)
	}
	return res, nil
}

// GetOpenPRsByReviewers находит открытые PR, где ревьюверами являются указанные пользователи.
// Возвращает мапу: ReviewerID -> Список PRID, где он назначен.
func (r *PRRepo) GetOpenPRsByReviewers(ctx context.Context, reviewerIDs []string) (map[string][]string, error) {
	q := r.db.GetQueryExecutor(ctx)

	rows, err := q.Query(ctx, `
		SELECT pr.pull_request_id, r.reviewer_id
		FROM pull_request_reviewers r
		JOIN pull_requests pr ON pr.pull_request_id = r.pull_request_id
		WHERE r.reviewer_id = ANY($1) AND pr.status = 'OPEN'
	`, reviewerIDs)
	if err != nil {
		return nil, fmt.Errorf("query impacted prs: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var prID, revID string
		if err := rows.Scan(&prID, &revID); err != nil {
			return nil, err
		}
		result[revID] = append(result[revID], prID)
	}
	return result, nil
}

// StatsDTO для статистики
type StatsDTO struct {
	ReviewerID  string `json:"reviewer_id"`
	ReviewCount int    `json:"review_count"`
}

// GetReviewerStats возвращает количество назначений по пользователям.
func (r *PRRepo) GetReviewerStats(ctx context.Context) ([]model.StatsDTO, error) {
	q := r.db.GetQueryExecutor(ctx)
	rows, err := q.Query(ctx, `
		SELECT reviewer_id, COUNT(*) 
		FROM pull_request_reviewers 
		GROUP BY reviewer_id 
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Используем тип из пакета model
	var stats []model.StatsDTO

	for rows.Next() {
		var s model.StatsDTO
		if err := rows.Scan(&s.ReviewerID, &s.ReviewCount); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return stats, nil
}

// RemoveReviewer удаляет ревьювера из PR.
// Используется, когда деактивированного пользователя некем заменить.
func (r *PRRepo) RemoveReviewer(ctx context.Context, prID, reviewerID string) error {
	q := r.db.GetQueryExecutor(ctx)

	_, err := q.Exec(ctx, `
		DELETE FROM pull_request_reviewers
		WHERE pull_request_id = $1 AND reviewer_id = $2
	`, prID, reviewerID)

	if err != nil {
		return fmt.Errorf("remove reviewer: %w", err)
	}
	return nil
}
