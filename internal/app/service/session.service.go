package service

import (
	"context"
	

	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/arnokay/arnobot-shared/db"
	"github.com/arnokay/arnobot-shared/storage"
	"github.com/google/uuid"
)

type SessionService struct {
	storage storage.Storager
	logger  applog.Logger
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

func (s *SessionService) DeleteOld(ctx context.Context, userID uuid.UUID) error {
	err := s.storage.Query(ctx).AuthSessionOldDeactivate(ctx, db.AuthSessionOldDeactivateParams{
		UserID: userID,
		Offset: 5,
	})
	if err != nil {
		return s.storage.HandleErr(ctx, err)
	}
	return nil
}

func (s *SessionService) Create(ctx context.Context, userID uuid.UUID) (*data.AuthSession, error) {
	fromDB, err := s.storage.Query(ctx).AuthSessionCreate(ctx, userID)
	if err != nil {
		return nil, s.storage.HandleErr(ctx, err)
	}

	session := data.NewSessionFromDB(fromDB)

	return &session, nil
}

func (s *SessionService) IsValidToken(ctx context.Context, token string) (bool, error) {
	status, err := s.storage.Query(ctx).AuthSessionValidate(ctx, token)
	if err != nil {
		return false, s.storage.HandleErr(ctx, err)
	}

	if status == db.AuthSessionStatusActive {
		return true, nil
	}

	return false, nil
}

func (s *SessionService) GetTokenOwner(ctx context.Context, token string) (*data.User, error) {
	fromDB, err := s.storage.Query(ctx).AuthSessionGetOwner(ctx, token)
	if err != nil {
		return nil, s.storage.HandleErr(ctx, err)
	}

	user := data.NewUserFromDB(fromDB.User)

	return &user, nil
}
