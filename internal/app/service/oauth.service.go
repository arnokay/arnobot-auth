package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/platform"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/thanhpk/randstr"
	"golang.org/x/oauth2"

	"github.com/arnokay/arnobot-auth/internal/app"
	"github.com/arnokay/arnobot-auth/internal/app/config"
)

type OAuthProvider interface {
	GetAuthURL(ctx context.Context, platform platform.Platform, state string) (string, error)
	CreateState(userID uuid.UUID) string
	ParseState(state string) uuid.UUID
	GetOAuthConfig(platform.Platform) (*oauth2.Config, error)
	HandleCallback(ctx context.Context, platform platform.Platform, code, state string) (*oauth2.Token, error)
}

type CacheEntry struct {
	State         string
	CodeVerifier  string
	CodeChallenge string
}

type OAuthService struct {
	cache jetstream.KeyValue

	logger *slog.Logger
}

type providerConfig struct {
	authURL       string
	tokenURL      string
	defaultScopes []string
	usePKCE       bool
}

var providerConfigs = map[platform.Platform]providerConfig{
	platform.Twitch: {
		authURL:       "https://id.twitch.tv/oauth2/authorize",
		tokenURL:      "https://id.twitch.tv/oauth2/token",
		defaultScopes: app.TwitchScopes,
		usePKCE:       false,
	},
	// "kick": {
	// 	authURL:       "https://kick.com/oauth2/authorize",
	// 	tokenURL:      "https://kick.com/oauth2/token",
	// 	defaultScopes: []string{"user:read"},
	// 	usePKCE:       true,
	// },
}

func NewOAuthService(
	cache jetstream.KeyValue,
) *OAuthService {
	logger := applog.NewServiceLogger("oauth-service")

	return &OAuthService{
		cache: cache,

		logger: logger,
	}
}

func (o *OAuthService) generateRandomString(length int) string {
	return randstr.Base62(length)
}

func (o *OAuthService) generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

func (o *OAuthService) getProviderConfig(platform platform.Platform) (providerConfig, error) {
	providerCfg, ok := providerConfigs[platform]
	if !ok {
		return providerConfig{}, apperror.ErrNotImplemented
	}
	return providerCfg, nil
}

func (o *OAuthService) GetOAuthConfig(p platform.Platform) (*oauth2.Config, error) {
	providerCfg, err := o.getProviderConfig(p)
	if err != nil {
		return nil, err
	}
	providerCreds, ok := config.Config.Providers[p]
	if !ok {
		return nil, apperror.ErrNotImplemented
	}
	config := &oauth2.Config{
		ClientID:     providerCreds.ClientID,
		ClientSecret: providerCreds.ClientSecret,
		RedirectURL:  providerCreds.RedirectURI,
		Scopes:       providerCfg.defaultScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  providerCfg.authURL,
			TokenURL: providerCfg.tokenURL,
		},
	}

	return config, nil
}

func (o *OAuthService) CreateState(userID uuid.UUID) string {
	parts := []string{o.generateRandomString(42)}
	if userID != uuid.Nil {
		parts = append(parts, userID.String())
	}

	return strings.Join(parts, "|||")
}

func (o *OAuthService) ParseState(state string) uuid.UUID {
	parts := strings.Split(state, "|||")
	if len(parts) > 1 {
		userID, err := uuid.Parse(parts[1])
		if err != nil {
			return uuid.Nil
		}
		return userID
	}

	return uuid.Nil
}

// GetAuthURL generates auth url using github.com/golang/x/oauth2 package
//
// If state is not provided, its gonna generate random state
func (o *OAuthService) GetAuthURL(ctx context.Context, platform platform.Platform, state string) (string, error) {
	providerCfg, err := o.getProviderConfig(platform)
	if err != nil {
		o.logger.DebugContext(ctx, "provider config doesnt exist", "platform", platform)
		return "", err
	}
	oauthCfg, err := o.GetOAuthConfig(platform)
	if err != nil {
		o.logger.DebugContext(ctx, "oauth config doesnt exist", "platform", platform)
		return "", err
	}

	if state == "" {
		state = o.generateRandomString(42)
	}

	entry := &CacheEntry{
		State: state,
	}

	var authCodeOptions []oauth2.AuthCodeOption

	if providerCfg.usePKCE {
		verifier := o.generateRandomString(43) // 43 is recommended length

		entry.CodeVerifier = verifier
		entry.CodeChallenge = o.generateCodeChallenge(verifier)

		authCodeOptions = append(authCodeOptions,
			oauth2.SetAuthURLParam("code_challenge", entry.CodeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}

	entryBytes, _ := json.Marshal(entry)
	_, err = o.cache.Create(ctx, state, entryBytes)
	if err != nil {
		o.logger.ErrorContext(ctx, "cannot cache state", "err", err, "entry", entry)
		return "", apperror.ErrInternal
	}

	authURL := oauthCfg.AuthCodeURL(state, authCodeOptions...)
	return authURL, nil
}

func (o *OAuthService) HandleCallback(
	ctx context.Context,
	platform platform.Platform,
	code,
	state string,
) (*oauth2.Token, error) {
	providerCfg, err := o.getProviderConfig(platform)
	if err != nil {
		o.logger.DebugContext(ctx, "provider config doesnt exist", "platform", platform)
		return nil, err
	}
	oauthCfg, err := o.GetOAuthConfig(platform)
	if err != nil {
		o.logger.DebugContext(ctx, "oauth config doesnt exist", "platform", platform)
		return nil, err
	}

	if code == "" {
		o.logger.DebugContext(ctx, "no code provided")
		return nil, apperror.ErrInvalidInput
	}
	if state == "" {
		o.logger.DebugContext(ctx, "no state provided")
		return nil, apperror.ErrInvalidInput
	}

	entryBytes, err := o.cache.Get(ctx, state)
	if err != nil {
		o.logger.DebugContext(ctx, "cannot find state in cache", "state", state)
		return nil, apperror.ErrNotFound
	}

	var entry CacheEntry
	err = json.Unmarshal(entryBytes.Value(), &entry)
	if err != nil {
		o.logger.ErrorContext(ctx, "cannot unmarshal state cache", "err", err, "state", state, "entry_bytes", entryBytes)
		return nil, apperror.ErrInternal
	}

	err = o.cache.Purge(ctx, state)
	if err != nil {
		o.logger.ErrorContext(ctx, "cannot purge state from cache", "err", err, "state", state)
		return nil, apperror.ErrInternal
	}

	var tokenOptions []oauth2.AuthCodeOption

	if providerCfg.usePKCE && entry.CodeVerifier != "" {
		tokenOptions = append(tokenOptions,
			oauth2.SetAuthURLParam("code_verifier", entry.CodeVerifier),
		)
	}

	token, err := oauthCfg.Exchange(ctx, code, tokenOptions...)
	if err != nil {
		o.logger.DebugContext(ctx, "failed to exchange code for token", "err", err)
		return nil, apperror.ErrExternal
	}

	if rawScope := token.Extra("scope"); rawScope != nil {
		if rawScopeSlice, ok := rawScope.([]any); ok {
			scopes := make([]string, 0, len(rawScopeSlice))
			for _, scope := range rawScopeSlice {
				if scopeStr, ok := scope.(string); ok {
					scopes = append(scopes, scopeStr)
				}
			}

			if !app.IsScopesEqual(app.TwitchScopes, scopes) {
				o.logger.WarnContext(
					ctx,
					"invalid scopes provided",
					"app_scopes", app.TwitchScopes,
					"scopes", scopes,
				)
				return nil, apperror.ErrInvalidInput
			}
		}
	}

	return token, nil
}
