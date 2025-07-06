package api

import (
	"errors"
	
	"net/http"
	"net/url"

	"github.com/arnokay/arnobot-shared/appctx"
	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/arnokay/arnobot-shared/platform"
	sharedService "github.com/arnokay/arnobot-shared/service"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/arnokay/arnobot-auth/internal/app/config"
	"github.com/arnokay/arnobot-auth/internal/app/service"
)

type providerController struct {
	platformAPIService *service.PlatformAPIService
	providerService    *service.AuthProviderService
	userService        *service.UserService
	sessionService     *service.SessionService
	transactionService sharedService.ITransactionService
	whitelistService   *service.WhitelistService
	oauthService       service.OAuthProvider

	logger applog.Logger
}

func NewProviderController(
	platformAPIService *service.PlatformAPIService,
	userService *service.UserService,
	providerService *service.AuthProviderService,
	sessionService *service.SessionService,
	transactionService sharedService.ITransactionService,
	whitelistService *service.WhitelistService,
	oauthService service.OAuthProvider,
) *providerController {
	logger := applog.NewServiceLogger("provider-controller")

	return &providerController{
		platformAPIService: platformAPIService,
		providerService:    providerService,
		userService:        userService,
		sessionService:     sessionService,
		transactionService: transactionService,
		whitelistService:   whitelistService,
		oauthService:       oauthService,
		logger:             logger,
	}
}

func (c *providerController) Routes(parentGroup *echo.Group) {
	group := parentGroup.Group("/provider")
	group.GET("/:platform", c.AuthURL)
	group.GET("/:platform/callback", c.Callback)
}

func (c *providerController) AuthURL(ctx echo.Context) error {
	reqCtx := ctx.Request().Context()

	var payload struct {
		Platform platform.Platform `param:"platform" validate:"validateFn"`
	}

	err := ctx.Bind(&payload)
	if err != nil {
		c.logger.DebugContext(reqCtx, "cannot bind payload", "err", err)
		return apperror.ErrInvalidInput
	}

	err = ctx.Validate(payload)
	if err != nil {
		c.logger.DebugContext(ctx.Request().Context(), "failed validation", "err", err)
		return err
	}

	var userID uuid.UUID

	user := appctx.GetUser(reqCtx)
	if user != nil {
		userID = user.ID
	}

	state := c.oauthService.CreateState(userID)
	url, err := c.oauthService.GetAuthURL(reqCtx, payload.Platform, state)
	if err != nil {
		return err
	}

	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (c *providerController) Callback(ctx echo.Context) error {
	reqCtx := ctx.Request().Context()

	var payload struct {
		Code     string            `query:"code" validate:"required"`
		State    string            `query:"state" validate:"required"`
		Platform platform.Platform `param:"platform" validate:"validateFn"`
	}

	err := ctx.Bind(&payload)
	if err != nil {
		c.logger.DebugContext(reqCtx, "cannot bind payload", "err", err)
		return apperror.ErrInvalidInput
	}

	err = ctx.Validate(payload)
	if err != nil {
		c.logger.DebugContext(ctx.Request().Context(), "failed validation", "err", err)
		return err
	}

	frontEndURL, _ := url.Parse(config.Config.FrontEndCallback)
	queryParams := frontEndURL.Query()

	token, err := c.oauthService.HandleCallback(reqCtx, payload.Platform, payload.Code, payload.State)
	if err != nil {
		return err
	}
	oauthCfg, err := c.oauthService.GetOAuthConfig(payload.Platform)
	if err != nil {
		return err
	}

	platformUser, err := c.platformAPIService.GetUserInfoFromToken(
		reqCtx,
		payload.Platform,
		oauthCfg,
		token,
	)
	if err != nil {
		return err
	}

	userID := c.oauthService.ParseState(payload.State)

	var whitelistID int32

	if config.Config.WhitelistEnabled {
		whitelist, err := c.whitelistService.GetOne(reqCtx, data.WhitelistGetOne{
			Platform:          payload.Platform,
			UserID:            &userID,
			PlatformUserID:    &platformUser.ID,
			PlatformUserName:  &platformUser.Name,
			PlatformUserLogin: &platformUser.Login,
		})
		if err != nil {
			c.logger.DebugContext(reqCtx, "user is not whitelisted, rejecting", "whitelist", whitelist)
			return err
		}
		whitelistID = whitelist.ID
	}

	txCtx, err := c.transactionService.Begin(reqCtx)
	if err != nil {
		return err
	}
	defer c.transactionService.Rollback(txCtx)

	provider, err := c.providerService.GetByProviderUserID(txCtx, platformUser.ID, payload.Platform.String())
	if err != nil {
		if !errors.Is(err, apperror.ErrNotFound) {
			return err
		}
	}
	if provider != nil {
		c.logger.DebugContext(txCtx, "provider already exists, updating tokens", "providerID", provider.ID)
		err = c.providerService.UpdateTokens(txCtx, data.AuthProviderUpdateTokens{
			ID:           provider.ID,
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
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
			userID, err = c.userService.CreateUser(txCtx, platformUser.Login)
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
			Provider:       payload.Platform.String(),
			ProviderUserID: platformUser.ID,
		})
		if err != nil {
			return err
		}
		c.logger.DebugContext(txCtx, "created provider", "providerID", providerID)
	}

	if config.Config.WhitelistEnabled && whitelistID != 0 {
		_, err := c.whitelistService.UpdateByID(txCtx, whitelistID, data.WhitelistUpdate{
			Platform:          &payload.Platform,
			UserID:            &userID,
			PlatformUserID:    &platformUser.ID,
			PlatformUserName:  &platformUser.Name,
			PlatformUserLogin: &platformUser.Login,
		})
		if err != nil {
			c.logger.DebugContext(reqCtx, "cannot update whitelist")
			return err
		}
	}

	c.logger.DebugContext(txCtx, "creating session", "userID", userID)
	session, err := c.sessionService.Create(txCtx, userID)
	if err != nil {
		return err
	}

	err = c.sessionService.DeleteOld(txCtx, userID)
	if err != nil {
		c.logger.DebugContext(txCtx, "cannot delete old sessions", "userID", userID)
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
