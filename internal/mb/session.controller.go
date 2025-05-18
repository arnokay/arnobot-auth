package mb

import (
	"log/slog"

	"arnobot-shared/applog"
	"arnobot-shared/mbtypes"
	"arnobot-shared/topics"

	"github.com/nats-io/nats.go"

	"arnobot-auth/internal/app/service"
)

type SessionController struct {
	sessionService *service.SessionService

	mb     *nats.Conn
	logger *slog.Logger
}

func NewSessionController(mb *nats.Conn, sessionService *service.SessionService) *SessionController {
	logger := applog.NewServiceLogger("mb-session-controller")

	return &SessionController{
		mb:             mb,
		logger:         logger,
		sessionService: sessionService,
	}
}

func (c *SessionController) Connect() {
  c.mb.QueueSubscribe(topics.AuthSessionTokenExchange, topics.AuthSessionTokenExchange, c.ExchangeTokenForUser)
	c.mb.QueueSubscribe(topics.AuthSessionTokenValidate, topics.AuthSessionTokenValidate, c.Validate)
}

func (c *SessionController) ExchangeTokenForUser(msg *nats.Msg) {
	var req mbtypes.AuthSessionTokenRequest
  var res mbtypes.AuthSessionTokenExchangeResponse

	req.Decode(msg.Data)

  ctx, cancel := newControllerContext(req.TraceID)
  defer cancel()

  user, err := c.sessionService.GetTokenOwner(ctx, req.Data)
  if err != nil {
    c.logger.DebugContext(ctx, "#exchange: get token owner error", "err", err)
    res.ToFail(err.Error())
    b, _ := res.Encode()
    msg.Respond(b)
    return
  }

  res.ToSuccess(user)
  b, _ := res.Encode()
  msg.Respond(b)
}

func (c *SessionController) Validate(msg *nats.Msg) {
	var req mbtypes.AuthSessionTokenRequest
  var res mbtypes.AuthSessionTokenValidateResponse

	req.Decode(msg.Data)

  ctx, cancel := newControllerContext(req.TraceID)
  defer cancel()

  isValid, err := c.sessionService.IsValidToken(ctx, req.Data)
  if err != nil {
    res.ToFail(err.Error())
    b, _ := res.Encode()
    msg.Respond(b)
    return
  }

  res.ToSuccess(isValid)
  b, _ := res.Encode()
  msg.Respond(b)
}

