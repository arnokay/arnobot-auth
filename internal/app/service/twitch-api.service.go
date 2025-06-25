package service

import (
	"context"
	"log/slog"

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	assert "github.com/arnokay/arnobot-shared/pkg/assert/panic"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nicklaw5/helix/v2"

	"github.com/arnokay/arnobot-auth/internal/app/config"
)

type TwitchAPIService struct {
	cache  jetstream.KeyValue
	helix  *helix.Client
	logger *slog.Logger
	cfg    *config.ProviderConfig
}

func NewTwitchAPIService(
	cache jetstream.KeyValue,
) *TwitchAPIService {
	cfg, ok := config.Config.Providers["twitch"]
	assert.Assert(ok, "TwitchService: config is not loaded for \"twitch\" provider")

	client, err := helix.NewClient(&helix.Options{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  cfg.RedirectURI,
	})
	assert.NoError(err, "TwitchService: helix client error")
	logger := applog.NewServiceLogger("TwitchApiService")

	return &TwitchAPIService{
		cache:  cache,
		helix:  client,
		logger: logger,
		cfg:    cfg,
	}
}

func (s *TwitchAPIService) GetUserInfoFromAccessToken(ctx context.Context, accessToken string) (helix.User, error) {
	client, _ := helix.NewClientWithContext(ctx, &helix.Options{
		ClientID:        s.cfg.ClientID,
		UserAccessToken: accessToken,
	})

	users, err := client.GetUsers(nil)
	if err != nil {
		return helix.User{}, err
	}

	if len(users.Data.Users) != 1 {
		s.logger.Error("somehow we got more then one user", "users", users.Data.Users)
		return helix.User{}, apperror.ErrInternal
	}

	user := users.Data.Users[0]

	return user, nil
}

func (s *TwitchAPIService) GetUsers() {
	s.helix.GetUsers(&helix.UsersParams{})
}
