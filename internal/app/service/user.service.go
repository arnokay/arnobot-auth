package service

import (
	"context"
	"log/slog"

	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/storage"

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
		return uuid.UUID{}, apperror.ErrAlreadyExists
	}

	return id, nil
}
