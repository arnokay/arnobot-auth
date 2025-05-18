package service

import (
	"context"
	"log/slog"

	"arnobot-shared/pkg/errs"
	"arnobot-shared/applog"
	"arnobot-shared/data"
	"arnobot-shared/db"
)

type AuthProviderService struct {
	queries db.Querier
	logger  *slog.Logger
}

func NewAuthProviderService(queries db.Querier) *AuthProviderService {
	serviceName := "ProviderService"

  logger := applog.NewServiceLogger(serviceName)

	return &AuthProviderService{
		queries: queries,
		logger:  logger,
	}
}

func (s *AuthProviderService) Create(ctx context.Context, d data.AuthProviderCreate) (int, error) {
	id, err := s.queries.CreateAuthProvider(ctx, d.ToDB())
	if err != nil {
		return 0, errs.ErrAlreadyExists
	}

	return int(id), nil
}

func (s *AuthProviderService) Update(ctx context.Context, d data.AuthProviderUpdateTokens) error {
  count, err := s.queries.AuthProviderUpdateTokens(ctx, d.ToDB())
  if err != nil {
    return errs.ErrInternal
  }
  if count == 0 {
    return errs.ErrNotFound
  }

  return nil
}

func (s *AuthProviderService) GetByProviderUserId(ctx context.Context, id string, providerName string) (*data.AuthProvider, error) {
	dbProvider, err := s.queries.AuthProviderGetByProviderUserId(ctx, db.AuthProviderGetByProviderUserIdParams{
		ProviderUserID: id,
		Provider:       providerName,
	})
	if err != nil {
		return nil, errs.ErrNotFound
	}

	provider := data.NewProviderAuthFromDB(dbProvider)

	return &provider, nil
}
