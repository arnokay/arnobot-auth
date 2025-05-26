package service

import (
	"context"
	"log/slog"

	"arnobot-shared/applog"
	"arnobot-shared/pkg/errs"
	"arnobot-shared/storage"
)

type UserService struct {
	storage storage.Storager

	logger *slog.Logger
}

func NewUserService(store storage.Storager) *UserService {
	logger := applog.NewServiceLogger("UserService")

	return &UserService{
		logger:  logger,
		storage: store,
	}
}

func (s *UserService) CreateUser(ctx context.Context, username string) (int32, error) {
	id, err := s.storage.Query(ctx).UserCreate(ctx, username)
	if err != nil {
		s.logger.Warn("cannot create user", "err", err)
		return 0, errs.ErrAlreadyExists
	}

	return id, nil
}
