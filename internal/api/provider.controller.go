package api

import (
	"log/slog"
	"net/http"
	"net/url"

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
	twitchApiService   *service.TwitchApiService
	providerService    *service.AuthProviderService
	userService        *service.UserService
	sessionService     *service.SessionService
	transactionService sharedService.ITransactionService

	logger *slog.Logger
}

func NewProviderController(
	twitchApiService *service.TwitchApiService,
	userService *service.UserService,
	providerService *service.AuthProviderService,
	sessionService *service.SessionService,
	transactionService sharedService.ITransactionService,
) *providerController {
	logger := applog.NewServiceLogger("provider-controller")

	return &providerController{
		twitchApiService:   twitchApiService,
		providerService:    providerService,
		userService:        userService,
		sessionService:     sessionService,
		transactionService: transactionService,
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

	url := c.twitchApiService.GenerateAuthURL(reqCtx, nil)

	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (c *providerController) TwitchCallback(ctx echo.Context) error {
	reqCtx := ctx.Request().Context()

	error := ctx.QueryParam("error")
	if error != "" {
		desc := ctx.QueryParam("error_description")
		c.logger.WarnContext(reqCtx, "error retrieving code", "error", error, "desc", desc)

		return apperror.ErrInvalidInput
	}

	code := ctx.QueryParam("code")
	if code == "" {
		c.logger.WarnContext(reqCtx, "no code provided", "url", ctx.Request().RequestURI)
		return apperror.ErrInvalidInput
	}

	state := ctx.QueryParam("state")
	if state == "" {
		c.logger.WarnContext(reqCtx, "no state provided", "url", ctx.Request().RequestURI)
		return apperror.ErrInvalidInput
	}

	isStateExists := c.twitchApiService.IsStateExists(reqCtx, state)
	if !isStateExists {
		c.logger.WarnContext(reqCtx, "state does not exists", "state", state)

		return apperror.ErrInvalidInput
	}

	token, err := c.twitchApiService.ExchangeCode(reqCtx, code)
	if err != nil {
		c.logger.ErrorContext(reqCtx, "cannot exchange code", "err", err)
		return apperror.ErrInvalidInput
	}

  // TODO: check token.Scopes

	twitchUser, err := c.twitchApiService.GetUserInfoFromAccessToken(reqCtx, token.AccessToken)
	if err != nil {
		return err
	}

	var userID uuid.UUID

  txCtx, err := c.transactionService.Begin(reqCtx)
  if err != nil {
    return err
  }
  defer c.transactionService.Rollback(txCtx)

  // TODO: add database level errors handling so you could distinguish NotFound and Connection errors
	provider, _ := c.providerService.GetByProviderUserId(txCtx, twitchUser.ID, "twitch")
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
		c.logger.DebugContext(txCtx, "provider not found, creating user and provider")
		userID, err = c.userService.CreateUser(txCtx, twitchUser.Login)
		if err != nil {
			return err
		}

		c.logger.DebugContext(txCtx, "created user", "userID", userID)
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

  frontEndURL, _ := url.Parse(config.Config.FrontEndCallback)
  queryParams := frontEndURL.Query()
  queryParams.Set("session", session.Token)
  frontEndURL.RawQuery = queryParams.Encode()

  err = c.transactionService.Commit(txCtx)
  if err != nil {
    return err
  }

	// TODO: change to redirect to frontend
	return ctx.Redirect(http.StatusPermanentRedirect, frontEndURL.String())
}
