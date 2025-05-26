package service

import (
	"context"
	"log/slog"

	"arnobot-shared/applog"
	"arnobot-shared/data"
	"arnobot-shared/db"
	"arnobot-shared/pkg/errs"
	"arnobot-shared/storage"
)

type SessionService struct {
  storage storage.Storager
	logger  *slog.Logger
}

func NewSessionService(
  store storage.Storager,
) *SessionService {
	logger := applog.NewServiceLogger("session-service")

	return &SessionService{
		logger:  logger,
    storage: store,
	}
}

func (s *SessionService) Create(ctx context.Context, userID int32) (*data.AuthSession, error) {
  fromDB, err := s.storage.Query(ctx).AuthSessionCreate(ctx, userID)
  if err != nil {
    return nil, errs.ErrAlreadyExists
  }

  session := data.NewSessionFromDB(fromDB)
  
  return &session, nil
}

func (s *SessionService) IsValidToken(ctx context.Context, token string) (bool, error)  {
  status, err := s.storage.Query(ctx).AuthSessionValidate(ctx, token)
  if err != nil {
    return false, errs.ErrNotFound
  }

  if status == db.AuthSessionStatusActive {
    return true, nil
  }

  return false, nil
}

func (s *SessionService) GetTokenOwner(ctx context.Context, token string) (*data.User, error) {
  fromDB, err := s.storage.Query(ctx).AuthSessionGetOwner(ctx, token)
  if err != nil {
    return nil, errs.ErrNotFound
  }

  user := data.NewUserFromDB(fromDB.User)

  return &user, nil
}




















