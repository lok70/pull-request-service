package repository

import (
	"context"
	"errors"
	"fmt"

	"pull-request-service/internal/model"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrTeamExists   = errors.New("team already exists")
	ErrTeamNotFound = errors.New("team not found")
)

type TeamRepo struct {
	db *Postgres
}

func NewTeamRepo(db *Postgres) *TeamRepo {
	return &TeamRepo{db: db}
}

func (r *TeamRepo) CreateTeamWithMembers(ctx context.Context, t model.Team) (model.Team, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return model.Team{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			_ = tx.Commit(ctx)
		}
	}()

	var teamID int64
	err = tx.QueryRow(ctx, `INSERT INTO teams (team_name) VALUES ($1) RETURNING id`, t.TeamName).Scan(&teamID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// уникальное ограничение по team_name нарушено
			return model.Team{}, ErrTeamExists
		}
		return model.Team{}, fmt.Errorf("insert team: %w", err)
	}

	for _, m := range t.Members {
		_, err = tx.Exec(ctx, `
INSERT INTO users (user_id, username, team_id, is_active)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE
SET username = EXCLUDED.username,
    team_id  = EXCLUDED.team_id,
    is_active = EXCLUDED.is_active
`, m.UserID, m.Username, teamID, m.IsActive)
		if err != nil {
			return model.Team{}, fmt.Errorf("upsert user %s: %w", m.UserID, err)
		}
	}

	return t, nil
}

func (r *TeamRepo) GetTeamByName(ctx context.Context, name string) (model.Team, error) {
	rows, err := r.db.Pool.Query(ctx, `
SELECT t.team_name, u.user_id, u.username, u.is_active
FROM teams t
LEFT JOIN users u ON u.team_id = t.id
WHERE t.team_name = $1
ORDER BY u.user_id
`, name)
	if err != nil {
		return model.Team{}, fmt.Errorf("query team: %w", err)
	}
	defer rows.Close()

	team := model.Team{
		TeamName: name,
		Members:  make([]model.TeamMember, 0),
	}

	foundTeam := false

	for rows.Next() {
		foundTeam = true

		var teamName string
		var userID *string
		var username *string
		var isActive *bool

		if err := rows.Scan(&teamName, &userID, &username, &isActive); err != nil {
			return model.Team{}, fmt.Errorf("scan row: %w", err)
		}

		team.TeamName = teamName

		if userID != nil && username != nil && isActive != nil {
			team.Members = append(team.Members, model.TeamMember{
				UserID:   *userID,
				Username: *username,
				IsActive: *isActive,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return model.Team{}, fmt.Errorf("rows error: %w", err)
	}

	if !foundTeam {
		return model.Team{}, ErrTeamNotFound
	}

	return team, nil
}
