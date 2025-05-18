package service

import (
	"context"
	"log/slog"

	"arnobot-shared/pkg/errs"
	"arnobot-shared/applog"
	"arnobot-shared/data"
	"arnobot-shared/db"
)

type TwitchService struct {
	queries db.Querier
	logger  *slog.Logger
}

func NewTwitchService(queries db.Querier) *TwitchService {
	logger := applog.NewServiceLogger("TwitchService")

	return &TwitchService{
		logger:  logger,
		queries: queries,
	}
}

func (s *TwitchService) Create(ctx context.Context, user data.TwitchUserCreate) (string, error) {
	id, err := s.queries.TwitchUserCreate(ctx, user.ToDB())
	if err != nil {
		return "", errs.ErrAlreadyExists
	}

	return id, nil
}
