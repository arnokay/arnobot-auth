package service

import (
	"context"
	"log/slog"

	"arnobot-shared/pkg/errs"
	"arnobot-shared/applog"
	"arnobot-shared/db"
)

type UserService struct {
	queries db.Querier
	logger  *slog.Logger
}

func NewUserService(queries db.Querier) *UserService {
  logger := applog.NewServiceLogger("UserService")

	return &UserService{
		logger:  logger,
		queries: queries,
	}
}

func (s *UserService) CreateUser(ctx context.Context) (int, error) {
	id, err := s.queries.UserCreate(ctx)
	if err != nil {
		s.logger.Warn("cannot create user", "err", err)
		return 0, errs.ErrAlreadyExists
	}

	return int(id), nil
}
