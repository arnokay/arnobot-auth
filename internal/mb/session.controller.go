package mb

import (
	"log/slog"

	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/apptype"
	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/topics"

	"github.com/nats-io/nats.go"

	"github.com/arnokay/arnobot-auth/internal/app/service"
)

type SessionController struct {
	sessionService *service.SessionService

	logger *slog.Logger
}

func NewSessionController(sessionService *service.SessionService) *SessionController {
	logger := applog.NewServiceLogger("mb-session-controller")

	return &SessionController{
		logger:         logger,
		sessionService: sessionService,
	}
}

func (c *SessionController) Connect(conn *nats.Conn) {
	conn.QueueSubscribe(topics.AuthSessionTokenExchange, topics.AuthSessionTokenExchange, c.exchangeTokenForUser)
	conn.QueueSubscribe(topics.AuthSessionTokenValidate, topics.AuthSessionTokenValidate, c.validate)
}

func (c *SessionController) exchangeTokenForUser(msg *nats.Msg) {
	var req apptype.AuthSessionTokenRequest
	var res apptype.AuthSessionTokenExchangeResponse

	req.Decode(msg.Data)

	ctx, cancel := newControllerContext(req.TraceID)
	defer cancel()
  res.TraceID = req.TraceID

	user, err := c.sessionService.GetTokenOwner(ctx, req.Data)
	if err != nil {
		c.logger.DebugContext(ctx, "#exchange: get token owner error", "err", err)
		res.ToFail(apperror.CodeInvalidInput, err.Error())
		b, _ := res.Encode()
		msg.Respond(b)
		return
	}

	res.ToSuccess(user)
	b, _ := res.Encode()
	msg.Respond(b)
}

func (c *SessionController) validate(msg *nats.Msg) {
	var req apptype.AuthSessionTokenRequest
	var res apptype.AuthSessionTokenValidateResponse

	req.Decode(msg.Data)

	ctx, cancel := newControllerContext(req.TraceID)
	defer cancel()
  res.TraceID = req.TraceID

	isValid, err := c.sessionService.IsValidToken(ctx, req.Data)
	if err != nil {
		res.ToFail(apperror.CodeInvalidInput, err.Error())
		b, _ := res.Encode()
		msg.Respond(b)
		return
	}

	res.ToSuccess(isValid)
	b, _ := res.Encode()
	msg.Respond(b)
}
