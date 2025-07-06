package service

import (
	"context"
	

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/arnokay/arnobot-shared/db"
	"github.com/arnokay/arnobot-shared/storage"
)

type AuthProviderService struct {
	storage storage.Storager
	logger  applog.Logger
}

func NewAuthProviderService(store storage.Storager) *AuthProviderService {
	serviceName := "provider-service"

	logger := applog.NewServiceLogger(serviceName)

	return &AuthProviderService{
		storage: store,
		logger:  logger,
	}
}

func (s *AuthProviderService) Create(ctx context.Context, arg data.AuthProviderCreate) (int, error) {
	id, err := s.storage.Query(ctx).AuthProviderCreate(ctx, arg.ToDB())
	if err != nil {
		s.logger.WarnContext(ctx, "cannot create auth provider", "err", err)
		return 0, apperror.ErrAlreadyExists
	}

	return int(id), nil
}

func (s *AuthProviderService) UpdateTokens(ctx context.Context, arg data.AuthProviderUpdateTokens) error {
	count, err := s.storage.Query(ctx).AuthProviderUpdateTokens(ctx, arg.ToDB())
	if err != nil {
		s.logger.ErrorContext(ctx, "cannot update tokens", "err", err, "provider_id", arg.ID)
		return apperror.ErrInternal
	}
	if count == 0 {
		s.logger.DebugContext(ctx, "cannot find auth provider", "provider_id", arg.ID)
		return apperror.ErrNotFound
	}

	return nil
}

func (s *AuthProviderService) Get(ctx context.Context, arg data.AuthProviderGet) (*data.AuthProvider, error) {
	if arg.ProviderUserID != nil && arg.UserID != nil {
		s.logger.DebugContext(ctx, "error: tried to use providerUserID and userID", "arg", arg)
		return nil, apperror.ErrInvalidInput
	}
	dbProvider, err := s.storage.Query(ctx).AuthProviderGet(ctx, arg.ToDB())
	if err != nil {
		s.logger.DebugContext(ctx, "cannot find provider", "err", err, "arg", arg)
		return nil, s.storage.HandleErr(ctx, err)
	}

	provider := data.NewProviderAuthFromDB(dbProvider)

	return &provider, nil
}

func (s *AuthProviderService) GetByProviderUserID(ctx context.Context, id string, providerName string) (*data.AuthProvider, error) {
	dbProvider, err := s.storage.Query(ctx).AuthProviderGetByProviderUserId(
		ctx,
		db.AuthProviderGetByProviderUserIdParams{
			ProviderUserID: id,
			Provider:       providerName,
		})
	if err != nil {
		return nil, s.storage.HandleErr(ctx, err)
	}

	provider := data.NewProviderAuthFromDB(dbProvider)

	return &provider, nil
}
