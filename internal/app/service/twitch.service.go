package service

import (
	"context"
	"log/slog"

	"arnobot-shared/applog"
	"arnobot-shared/data"
	"arnobot-shared/pkg/errs"
	"arnobot-shared/storage"
)

type TwitchService struct {
	storage storage.Storager
	logger  *slog.Logger
}

func NewTwitchService(
	store storage.Storager,
) *TwitchService {
	logger := applog.NewServiceLogger("TwitchService")

	return &TwitchService{
		logger:  logger,
		storage: store,
	}
}

func (s *TwitchService) Create(ctx context.Context, user data.TwitchUserCreate) (string, error) {
	id, err := s.storage.Query(ctx).TwitchUserCreate(ctx, user.ToDB())
	if err != nil {
		return "", errs.ErrAlreadyExists
	}

	return id, nil
}
