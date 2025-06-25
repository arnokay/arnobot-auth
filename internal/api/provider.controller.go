package api

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/arnokay/arnobot-shared/appctx"
	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	sharedService "github.com/arnokay/arnobot-shared/service"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/arnokay/arnobot-auth/internal/app/config"
	"github.com/arnokay/arnobot-auth/internal/app/service"
)

type providerController struct {
	twitchAPIService   *service.TwitchAPIService
	providerService    *service.AuthProviderService
	userService        *service.UserService
	sessionService     *service.SessionService
	transactionService sharedService.ITransactionService
	whitelistService   *service.WhitelistService
	twitchOAuthService service.OAuthProvider

	logger *slog.Logger
}

func NewProviderController(
	twitchAPIService *service.TwitchAPIService,
	userService *service.UserService,
	providerService *service.AuthProviderService,
	sessionService *service.SessionService,
	transactionService sharedService.ITransactionService,
	whitelistService *service.WhitelistService,
	twitchOAuthService service.OAuthProvider,
) *providerController {
	logger := applog.NewServiceLogger("provider-controller")

	return &providerController{
		twitchAPIService:   twitchAPIService,
		providerService:    providerService,
		userService:        userService,
		sessionService:     sessionService,
		transactionService: transactionService,
		whitelistService:   whitelistService,
		twitchOAuthService: twitchOAuthService,
		logger:             logger,
	}
}

func (c *providerController) Routes(parentGroup *echo.Group) {
	group := parentGroup.Group("/provider")
	group.GET("/twitch", c.TwitchAuthURL)
	group.GET("/twitch/callback", c.TwitchCallback)
}

func (c *providerController) TwitchAuthURL(ctx echo.Context) error {
	reqCtx := ctx.Request().Context()

	var userID uuid.UUID

	user := appctx.GetUser(reqCtx)
	if user != nil {
		userID = user.ID
	}

	state := c.twitchOAuthService.CreateState(userID)
	url, err := c.twitchOAuthService.GetAuthURL(reqCtx, state)
	if err != nil {
		return err
	}

	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (c *providerController) TwitchCallback(ctx echo.Context) error {
	reqCtx := ctx.Request().Context()

	frontEndURL, _ := url.Parse(config.Config.FrontEndCallback)
	queryParams := frontEndURL.Query()

	code := ctx.QueryParam("code")
	state := ctx.QueryParam("state")

	token, err := c.twitchOAuthService.HandleCallback(reqCtx, code, state)
	if err != nil {
		return err
	}

	twitchUser, err := c.twitchAPIService.GetUserInfoFromAccessToken(reqCtx, token.AccessToken)
	if err != nil {
		return err
	}

	userID := c.twitchOAuthService.ParseState(state)

	if config.Config.WhitelistEnabled {
		whitelist, err := c.whitelistService.GetOne(reqCtx, data.WhitelistGetOne{
			Platform:          "twitch",
			UserID:            &userID,
			PlatformUserID:    &twitchUser.ID,
			PlatformUserName:  &twitchUser.DisplayName,
			PlatformUserLogin: &twitchUser.Login,
		})
		if err != nil {
			c.logger.DebugContext(reqCtx, "user is not whitelisted, rejecting", "whitelist", whitelist)
			return err
		}
	}

	txCtx, err := c.transactionService.Begin(reqCtx)
	if err != nil {
		return err
	}
	defer c.transactionService.Rollback(txCtx)

	provider, err := c.providerService.GetByProviderUserID(txCtx, twitchUser.ID, "twitch")
	if err != nil {
		if !errors.Is(err, apperror.ErrNotFound) {
			return err
		}
	}
	if provider != nil {
		c.logger.DebugContext(txCtx, "provider already exists, updating tokens", "providerID", provider.ID)
		err = c.providerService.UpdateTokens(txCtx, provider.ID, data.AuthProviderUpdateTokens{
			AccessToken:  token.AccessToken,
			RefreshToken: &token.RefreshToken,
		})
		if err != nil {
			c.logger.DebugContext(txCtx, "cannot update provider tokens")
			return err
		}
		userID = provider.UserID
	} else {
		c.logger.DebugContext(txCtx, "provider not found")
		if userID == uuid.Nil {
			c.logger.DebugContext(txCtx, "user id is nil, creating user")
			userID, err = c.userService.CreateUser(txCtx, twitchUser.Login)
			if err != nil {
				return err
			}
			c.logger.DebugContext(txCtx, "created user", "userID", userID)
		}

		c.logger.DebugContext(txCtx, "creating provider")
		providerID, err := c.providerService.Create(txCtx, data.AuthProviderCreate{
			UserID:         userID,
			AccessToken:    token.AccessToken,
			RefreshToken:   token.RefreshToken,
			AccessType:     token.TokenType,
			Provider:       "twitch",
			ProviderUserID: twitchUser.ID,
		})
		if err != nil {
			return err
		}
		c.logger.DebugContext(txCtx, "created provider", "providerID", providerID)
	}

	c.logger.DebugContext(txCtx, "creating session", "userID", userID)
	session, err := c.sessionService.Create(txCtx, userID)
	if err != nil {
		return err
	}

	queryParams.Set("session", session.Token)
	frontEndURL.RawQuery = queryParams.Encode()

	err = c.transactionService.Commit(txCtx)
	if err != nil {
		return err
	}

	// TODO: change to redirect to frontend
	return ctx.Redirect(http.StatusPermanentRedirect, frontEndURL.String())
}
