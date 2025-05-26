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

type AuthProviderService struct {
	storage storage.Storager
	logger  *slog.Logger
}

func NewAuthProviderService(store storage.Storager) *AuthProviderService {
	serviceName := "provider-service"

	logger := applog.NewServiceLogger(serviceName)

	return &AuthProviderService{
		storage: store,
		logger:  logger,
	}
}

func (s *AuthProviderService) Create(ctx context.Context, d data.AuthProviderCreate) (int, error) {
	id, err := s.storage.Query(ctx).AuthProviderCreate(ctx, d.ToDB())
	if err != nil {
		s.logger.WarnContext(ctx, "cannot create auth provider", "err", err)
		return 0, errs.ErrAlreadyExists
	}

	return int(id), nil
}

func (s *AuthProviderService) UpdateTokens(ctx context.Context, id int32, d data.AuthProviderUpdateTokens) error {
	count, err := s.storage.Query(ctx).AuthProviderUpdateTokens(ctx, d.ToDB(id))
	if err != nil {
    s.logger.ErrorContext(ctx, "cannot update tokens", "err", err, "provider_id", id)
		return errs.ErrInternal
	}
	if count == 0 {
    s.logger.DebugContext(ctx, "cannot find auth provider", "provider_id", id)
		return errs.ErrNotFound
	}

	return nil
}

func (s *AuthProviderService) GetByProviderUserId(ctx context.Context, id string, providerName string) (*data.AuthProvider, error) {
	dbProvider, err := s.storage.Query(ctx).AuthProviderGetByProviderUserId(ctx, db.AuthProviderGetByProviderUserIdParams{
		ProviderUserID: id,
		Provider:       providerName,
	})
	if err != nil {
		return nil, errs.ErrNotFound
	}

	provider := data.NewProviderAuthFromDB(dbProvider)

	return &provider, nil
}
