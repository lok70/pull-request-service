package repository

import (
	"context"
	"errors"
	"fmt"

	"pull-request-service/internal/model"

	"github.com/jackc/pgx/v5"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type UserRepo struct {
	db *Postgres
}

func NewUserRepo(db *Postgres) *UserRepo {
	return &UserRepo{db: db}
}

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
