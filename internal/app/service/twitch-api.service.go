package service

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"arnobot-shared/applog"
	"arnobot-shared/pkg"
	"arnobot-shared/pkg/assert/panic"
	"arnobot-shared/pkg/errs"

	"github.com/nicklaw5/helix/v2"
	"github.com/thanhpk/randstr"

	"arnobot-auth/internal/app"
	"arnobot-auth/internal/app/config"
)

type TwitchApiService struct {
	store  pkg.Cacher
	helix  *helix.Client
	logger *slog.Logger
	cfg    *config.ProviderConfig
}

func NewTwitchApiService(store pkg.Cacher) *TwitchApiService {
	cfg, ok := config.Config.Providers["twitch"]
	assert.Assert(ok, "TwitchService: config is not loaded for \"twitch\" provider")

	client, err := helix.NewClient(&helix.Options{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  cfg.RedirectURI,
	})
	assert.NoError(err, "TwitchService: helix client error")

	assert.Assert(store != nil, "TwitchService: store is nil")

	logger := applog.NewServiceLogger("TwitchApiService")

	return &TwitchApiService{
		store:  store,
		helix:  client,
		logger: logger,
		cfg:    cfg,
	}
}

func (s *TwitchApiService) GenerateAuthURL(ctx context.Context, userId *int) string {
	state := s.GenerateState(ctx, userId)
	err := s.StoreState(ctx, state)
	assert.NoError(err, "cannot store state")

	authURL := s.helix.GetAuthorizationURL(&helix.AuthorizationURLParams{
		ResponseType: "code",
		Scopes:       app.TWITCH_SCOPES,
		State:        state,
	})

	return authURL
}

func (s *TwitchApiService) ParseState(ctx context.Context, state string) *int {
	stateParts := strings.Split(state, ":")
	if len(stateParts) != 2 {
		return nil
	}

	id, err := strconv.Atoi(stateParts[1])
	assert.NoError(err, "state id is not a number")

	return &id
}

func (s *TwitchApiService) GenerateState(ctx context.Context, userId *int) string {
	// TODO: move 12 to const
	stateStart := randstr.Base62(12)

	var userIdStr string
	if userId != nil {
		userIdStr = strconv.Itoa(*userId)
	}

	stateParts := []string{stateStart}

	if userIdStr != "" {
		stateParts = append(stateParts, userIdStr)
	}

	return strings.Join(stateParts, ":")
}

func (s *TwitchApiService) StoreState(ctx context.Context, state string) error {
	err := s.store.Set(state, "")
	if err != nil {
		s.logger.Error("cannot store state", "state", state, "err", err)
		return errs.ErrInternal
	}

	return nil
}

func (s *TwitchApiService) IsStateExists(ctx context.Context, state string) bool {
	_, err := s.store.Get(state)
	if err != nil {
    s.logger.Error("store error", "err", err)
		return false
	}

	return true
}

func (s *TwitchApiService) ExchangeCode(ctx context.Context, code string) (ProviderToken, error) {
	token, err := s.helix.RequestUserAccessToken(code)
	if err != nil || token.ErrorMessage != "" {
		s.logger.WarnContext(ctx, "couldnt exchange code for a token", "code", code, "err", err, "errMsg", token.ErrorMessage)
		return ProviderToken{}, errs.ErrInternal
	}

	return ProviderToken{
		AccessToken:  token.Data.AccessToken,
		RefreshToken: token.Data.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Duration(token.Data.ExpiresIn) * time.Second),
	}, nil
}

func (s *TwitchApiService) GetUserInfoFromAccessToken(ctx context.Context, accessToken string) (helix.User, error) {
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
		return helix.User{}, errs.ErrInternal
	}

	user := users.Data.Users[0]

	return user, nil
}

func (s *TwitchApiService) GetUsers() {
	s.helix.GetUsers(&helix.UsersParams{})
}
