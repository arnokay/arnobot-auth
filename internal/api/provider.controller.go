package api

import (
	"log/slog"
	"net/http"

	"arnobot-shared/applog"
	"arnobot-shared/data"
	"arnobot-shared/pkg/errs"
	"github.com/labstack/echo/v4"

	"arnobot-auth/internal/app/service"
)

type providerController struct {
	twitchApiService *service.TwitchApiService
	providerService  *service.AuthProviderService
	userService      *service.UserService
	sessionService   *service.SessionService
	logger           *slog.Logger
}

func NewProviderController(
	twitchApiService *service.TwitchApiService,
	userService *service.UserService,
	providerService *service.AuthProviderService,
	sessionService *service.SessionService,
) *providerController {
	logger := applog.NewServiceLogger("provider-controller")

	return &providerController{
		twitchApiService: twitchApiService,
		providerService:  providerService,
		userService:      userService,
		sessionService:   sessionService,
		logger:           logger,
	}
}

func (c *providerController) Routes(parentGroup *echo.Group) {
	group := parentGroup.Group("/provider")
	group.GET("/twitch", c.TwitchAuthURL)
	group.GET("/twitch/callback", c.TwitchCallback)
}

func (c *providerController) TwitchAuthURL(ctx echo.Context) error {
  reqCtx := ctx.Request().Context()

	url := c.twitchApiService.GenerateAuthURL(reqCtx, nil)

	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (c *providerController) TwitchCallback(ctx echo.Context) error {
  reqCtx := ctx.Request().Context()

	error := ctx.QueryParam("error")
	if error != "" {
		desc := ctx.QueryParam("error_description")
		c.logger.WarnContext(reqCtx, "error retrieving code", "error", error, "desc", desc)

		return errs.ErrInvalidInput
	}

	code := ctx.QueryParam("code")
	if code == "" {
		c.logger.WarnContext(reqCtx, "no code provided", "url", ctx.Request().RequestURI)
		return errs.ErrInvalidInput
	}

	state := ctx.QueryParam("state")
	if state == "" {
		c.logger.WarnContext(reqCtx, "no state provided", "url", ctx.Request().RequestURI)
		return errs.ErrInvalidInput
	}

	isStateExists := c.twitchApiService.IsStateExists(reqCtx, state)
	if !isStateExists {
		c.logger.WarnContext(reqCtx, "state does not exists", "state", state)

		return errs.ErrInvalidInput
	}

	token, err := c.twitchApiService.ExchangeCode(reqCtx, code)
	if err != nil {
		c.logger.ErrorContext(reqCtx, "cannot exchange code", "err", err)
		return errs.ErrInvalidInput
	}

	twitchUser, err := c.twitchApiService.GetUserInfoFromAccessToken(reqCtx, token.AccessToken)
	if err != nil {
		return err
	}

	var userID int

	provider, _ := c.providerService.GetByProviderUserId(reqCtx, twitchUser.ID, "twitch")
	if provider != nil {
		c.logger.DebugContext(reqCtx, "provider already exists, updating tokens", "providerID", provider.ID)
		err = c.providerService.Update(reqCtx, data.AuthProviderUpdateTokens{
			ID:           provider.ID,
			AccessToken:  token.AccessToken,
			RefreshToken: &token.RefreshToken,
		})
		userID = provider.UserID
	} else {
		c.logger.DebugContext(reqCtx, "provider not found, creating user and provider")
		userID, err = c.userService.CreateUser(reqCtx)
		if err != nil {
			return err
		}
		c.logger.DebugContext(reqCtx, "created user", "userID", userID)
		providerID, err := c.providerService.Create(reqCtx, data.AuthProviderCreate{
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
		c.logger.DebugContext(reqCtx, "created provider", "providerID", providerID)
	}

	c.logger.DebugContext(reqCtx, "creating session", "userID", userID)
	session, err := c.sessionService.Create(reqCtx, userID)
	if err != nil {
		return err
	}
	// TODO: change to redirect to frontend
	return ctx.JSON(http.StatusOK, session)
}
