package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/arnokay/arnobot-shared/platform"
	"golang.org/x/oauth2"
)

type PlatformAPIAdapter interface {
	GetUserInfo(ctx context.Context, client *http.Client) (*data.PlatformUser, error)
	GetPlatform() platform.Platform
}

type PlatformAPIService struct {
	logger *slog.Logger
}

type clientIDTransport struct {
	ClientID string
	Base     http.RoundTripper
}

func (t *clientIDTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())

	newReq.Header.Set("Client-Id", t.ClientID)

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	return base.RoundTrip(newReq)
}

func NewPlatformAPIService() *PlatformAPIService {
	logger := applog.NewServiceLogger("platform-api-service")

	return &PlatformAPIService{
		logger: logger,
	}
}

func (s *PlatformAPIService) getClient(
	ctx context.Context,
	p platform.Platform,
	config *oauth2.Config,
	token *oauth2.Token,
) *http.Client {
	client := http.DefaultClient

	switch p {
	case platform.Twitch:
		client = &http.Client{
			Transport: &clientIDTransport{
				ClientID: config.ClientID,
				Base:     http.DefaultTransport,
			},
		}
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, client)

	client = config.Client(ctx, token)

	return client
}

func (s *PlatformAPIService) GetUserInfoFromToken(
	ctx context.Context,
	platform platform.Platform,
	config *oauth2.Config,
	token *oauth2.Token,
) (*data.PlatformUser, error) {
	client := s.getClient(ctx, platform, config, token)
	adapter := s.getAdapter(platform)
	if adapter == nil {
		s.logger.DebugContext(ctx, "unknown platform adapter", "platform", platform)
		return nil, apperror.ErrNotImplemented
	}

	user, err := adapter.GetUserInfo(ctx, client)
	if err != nil {
		s.logger.DebugContext(ctx, "cannot request user info", "err", err)
		return nil, apperror.ErrExternal
	}

	return user, nil
}

func (s *PlatformAPIService) GetUserInfoFromAccessToken(
	ctx context.Context,
	platform platform.Platform,
	config *oauth2.Config,
	accessToken string,
) (*data.PlatformUser, error) {
	token := &oauth2.Token{AccessToken: accessToken}
	return s.GetUserInfoFromToken(ctx, platform, config, token)
}

func (s *PlatformAPIService) getAdapter(p platform.Platform) PlatformAPIAdapter {
	switch p {
	case platform.Twitch:
		return &TwitchAdapter{}
	default:
		return nil
	}
}

type TwitchAdapter struct{}

func (a *TwitchAdapter) GetPlatform() platform.Platform {
	return platform.Twitch
}

func (a *TwitchAdapter) GetUserInfo(ctx context.Context, client *http.Client) (*data.PlatformUser, error) {
	resp, err := client.Get("https://api.twitch.tv/helix/users")
	if err != nil {
		return nil, fmt.Errorf("twitch api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twitch api returned status %d", resp.StatusCode)
	}

	var response struct {
		Data []struct {
			ID          string `json:"id"`
			Login       string `json:"login"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode twitch response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no user data returned from twitch")
	}

	user := response.Data[0]
	return &data.PlatformUser{
		ID:       user.ID,
		Login:    user.Login,
		Name:     user.DisplayName,
		Platform: a.GetPlatform(),
	}, nil
}
