package service

import (
	"context"
	"log/slog"

	"arnobot-shared/applog"
	"arnobot-shared/pkg/errs"
	"arnobot-shared/storage"

	"github.com/google/uuid"
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

func (s *UserService) CreateUser(ctx context.Context, username string) (uuid.UUID, error) {
	id, err := s.storage.Query(ctx).UserCreate(ctx, username)
	if err != nil {
		s.logger.WarnContext(ctx, "cannot create user", "err", err)
		return uuid.UUID{}, errs.ErrAlreadyExists
	}

	return id, nil
}
