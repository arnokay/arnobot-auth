package service

import (
	"context"
	"log/slog"

	"arnobot-shared/applog"
	"arnobot-shared/data"
	"arnobot-shared/db"
	"arnobot-shared/pkg/errs"
)

type SessionService struct {
	querier db.Querier
	logger  *slog.Logger
}

func NewSessionService(
	querier db.Querier,
) *SessionService {
	logger := applog.NewServiceLogger("session-service")

	return &SessionService{
		logger:  logger,
		querier: querier,
	}
}

func (s *SessionService) Create(ctx context.Context, userID int) (*data.AuthSession, error) {
  fromDB, err := s.querier.AuthSessionCreate(ctx, int32(userID))
  if err != nil {
    return nil, errs.ErrAlreadyExists
  }

  session := data.NewSessionFromDB(fromDB)
  
  return &session, nil
}

func (s *SessionService) IsValidToken(ctx context.Context, token string) (bool, error)  {
  status, err := s.querier.AuthSessionValidate(ctx, token)
  if err != nil {
    return false, errs.ErrNotFound
  }

  if status == db.AuthSessionStatusActive {
    return true, nil
  }

  return false, nil
}

func (s *SessionService) GetTokenOwner(ctx context.Context, token string) (*data.User, error) {
  fromDB, err := s.querier.AuthSessionGetOwner(ctx, token)
  if err != nil {
    return nil, errs.ErrNotFound
  }

  user := data.NewUserFromDB(fromDB.User)

  return &user, nil
}




















