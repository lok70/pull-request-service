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

var (
	ErrPRNotFound = errors.New("pull request not found")
	ErrPRExists   = errors.New("pull request already exists")
)

type PRRepo struct {
	db *Postgres
}

func NewPRRepo(db *Postgres) *PRRepo {
	return &PRRepo{db: db}
}

func (r *PRRepo) CreatePRWithReviewers(ctx context.Context, pr model.PullRequest, reviewerIDs []string) (model.PullRequest, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return model.PullRequest{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			_ = tx.Commit(ctx)
		}
	}()

	row := tx.QueryRow(ctx, `
INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
VALUES ($1, $2, $3, $4)
RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
`, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, string(pr.Status))

	var created model.PullRequest
	var status string
	var createdAt time.Time
	var mergedAt *time.Time

	if err = row.Scan(&created.PullRequestID, &created.PullRequestName, &created.AuthorID, &status, &createdAt, &mergedAt); err != nil {
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
		br := tx.SendBatch(ctx, batch)
		if err = br.Close(); err != nil {
			return model.PullRequest{}, fmt.Errorf("insert reviewers: %w", err)
		}
	}

	reviewers, err := r.listReviewers(ctx, tx, created.PullRequestID)
	if err != nil {
		return model.PullRequest{}, err
	}
	created.AssignedReviewers = reviewers

	return created, nil
}

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

type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}
