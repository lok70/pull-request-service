package service

import (
	"context"
	"errors"
	"math/rand"

	"pull-request-service/internal/model"
	"pull-request-service/internal/repository"
)

// TeamRepository описывает контракт репозитория команд для бизнес-слоя.
type TeamRepository interface {
	CreateTeamWithMembers(ctx context.Context, team model.Team) (model.Team, error)
	GetTeamByName(ctx context.Context, name string) (model.Team, error)
}

// TeamService содержит бизнес-логику по созданию и получению команд.
type TeamService struct {
	repo      TeamRepository
	userRepo  UserRepository // <-- Добавили
	prRepo    PRRepository   // <-- Добавили
	txManager TransactionManager
}

// NewTeamService создаёт новый сервис для операций над командами.
func NewTeamService(
	repo TeamRepository,
	userRepo UserRepository,
	prRepo PRRepository,
	txManager TransactionManager,
) *TeamService {
	return &TeamService{
		repo:      repo,
		userRepo:  userRepo,
		prRepo:    prRepo,
		txManager: txManager,
	}
}

// CreateTeam валидирует входные данные и создаёт команду с участниками.
// В случае конфликтов по имени команды возвращает доменную ошибку TEAM_EXISTS.
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

// GetTeam возвращает команду по имени вместе с её участниками.
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

// MassDeactivate деактивирует пользователей и переназначает/удаляет их из открытых PR.
// MassDeactivate деактивирует пользователей и безопасно обновляет PR.
func (s *TeamService) MassDeactivate(ctx context.Context, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	return s.txManager.RunInTransaction(ctx, func(ctx context.Context) error {
		if err := s.userRepo.DeactivateUsers(ctx, userIDs); err != nil {
			return err
		}

		impactedPRsMap, err := s.prRepo.GetOpenPRsByReviewers(ctx, userIDs)
		if err != nil {
			return err
		}

		if len(impactedPRsMap) == 0 {
			return nil
		}

		for oldUserID, prIDs := range impactedPRsMap {
			oldUser, err := s.userRepo.GetByUserID(ctx, oldUserID)
			if err != nil {
				return err
			}

			for _, prID := range prIDs {
				pr, err := s.prRepo.GetPR(ctx, prID)
				if err != nil {
					return err
				}

				exclude := []string{oldUserID, pr.AuthorID}
				exclude = append(exclude, pr.AssignedReviewers...)

				candidates, err := s.userRepo.ListActiveTeamMembersExcept(ctx, oldUser.TeamName, exclude)
				if err != nil {
					return err
				}

				if len(candidates) > 0 {
					newReviewer := candidates[rand.Intn(len(candidates))]
					if _, err := s.prRepo.ReassignReviewer(ctx, prID, oldUserID, newReviewer.UserID); err != nil {
						return err
					}
				} else {
					if err := s.prRepo.RemoveReviewer(ctx, prID, oldUserID); err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
}

// GetStats возвращает статистику (прокси метод)
func (s *TeamService) GetStats(ctx context.Context) (interface{}, error) {
	// Для простоты возвращаем DTO репозитория, но по хорошему надо мапить в модель
	return s.prRepo.GetReviewerStats(ctx)
}
