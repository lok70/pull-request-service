package repository

import (
	"context"
	"errors"
	"fmt"

	"pull-request-service/internal/model"

	"github.com/jackc/pgx/v5"
)

// UserRepo реализует репозиторий пользователей на базе PostgreSQL.
type UserRepo struct {
	db *Postgres
}

// NewUserRepo создаёт новый экземпляр UserRepo c переданным подключением к PostgreSQL.
func NewUserRepo(db *Postgres) *UserRepo {
	return &UserRepo{db: db}
}

// GetByUserID возвращает пользователя по user_id вместе с именем его команды.
// Если пользователь не найден, возвращает ErrUserNotFound.
func (r *UserRepo) GetByUserID(ctx context.Context, userID string) (model.User, error) {
	row := r.db.Pool.QueryRow(ctx, `
SELECT u.user_id, u.username, t.team_name, u.is_active
FROM users u
JOIN teams t ON u.team_id = t.id
WHERE u.user_id = $1
`, userID)

	var u model.User
	if err := row.Scan(&u.UserID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

// SetIsActive обновляет флаг активности пользователя и возвращает обновлённого пользователя
// вместе с именем его команды. Если пользователь не найден, возвращает ErrUserNotFound.
func (r *UserRepo) SetIsActive(ctx context.Context, userID string, isActive bool) (model.User, error) {
	row := r.db.Pool.QueryRow(ctx, `
UPDATE users u
SET is_active = $2
FROM teams t
WHERE u.user_id = $1 AND u.team_id = t.id
RETURNING u.user_id, u.username, t.team_name, u.is_active
`, userID, isActive)

	var u model.User
	if err := row.Scan(&u.UserID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("update user: %w", err)
	}
	return u, nil
}

// ListActiveTeamMembersExcept возвращает список активных участников команды по её имени,
// исключая переданные user_id (exclude). Используется для выбора кандидатов в ревьюверы.
func (r *UserRepo) ListActiveTeamMembersExcept(ctx context.Context, teamName string, exclude []string) ([]model.User, error) {
	rows, err := r.db.Pool.Query(ctx, `
SELECT u.user_id, u.username, t.team_name, u.is_active
FROM users u
JOIN teams t ON u.team_id = t.id
WHERE t.team_name = $1 AND u.is_active = TRUE
ORDER BY u.user_id
`, teamName)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	excludeSet := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		excludeSet[id] = struct{}{}
	}

	users := make([]model.User, 0)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.UserID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if _, skip := excludeSet[u.UserID]; skip {
			continue
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return users, nil
}

// DeactivateUsers массово деактивирует пользователей по списку ID.
func (r *UserRepo) DeactivateUsers(ctx context.Context, userIDs []string) error {
	q := r.db.GetQueryExecutor(ctx)
	_, err := q.Exec(ctx, `
		UPDATE users 
		SET is_active = FALSE 
		WHERE user_id = ANY($1)
	`, userIDs)
	if err != nil {
		return fmt.Errorf("mass deactivate: %w", err)
	}
	return nil
}
