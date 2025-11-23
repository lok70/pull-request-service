package service

import (
	"context"
	"errors"

	"pull-request-service/internal/model"
	"pull-request-service/internal/repository"
)

type TeamRepository interface {
	CreateTeamWithMembers(ctx context.Context, team model.Team) (model.Team, error)
	GetTeamByName(ctx context.Context, name string) (model.Team, error)
}

type TeamService struct {
	repo TeamRepository
}

func NewTeamService(repo TeamRepository) *TeamService {
	return &TeamService{repo: repo}
}

func (s *TeamService) CreateTeam(ctx context.Context, t model.Team) (model.Team, error) {
	if t.TeamName == "" {
		return model.Team{}, ErrBadRequest("team_name must not be empty")
	}
	if len(t.Members) == 0 {
		return model.Team{}, ErrBadRequest("members must not be empty")
	}

	team, err := s.repo.CreateTeamWithMembers(ctx, t)
	if err != nil {
		if errors.Is(err, repository.ErrTeamExists) {
			return model.Team{}, ErrDomain("TEAM_EXISTS", "team_name already exists")
		}
		return model.Team{}, &AppError{
			Code:    "INTERNAL",
			Message: "failed to create team",
			Status:  500,
			Err:     err,
		}
	}
	return team, nil
}

func (s *TeamService) GetTeam(ctx context.Context, name string) (model.Team, error) {
	if name == "" {
		return model.Team{}, ErrBadRequest("team_name is required")
	}
	team, err := s.repo.GetTeamByName(ctx, name)
	if err != nil {
		if errors.Is(err, repository.ErrTeamNotFound) {
			return model.Team{}, ErrNotFound("team not found")
		}
		return model.Team{}, &AppError{
			Code:    "INTERNAL",
			Message: "failed to get team",
			Status:  500,
			Err:     err,
		}
	}
	return team, nil
}
