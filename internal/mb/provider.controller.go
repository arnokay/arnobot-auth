package mb

import (
	"log/slog"

	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/apptype"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/arnokay/arnobot-shared/topics"
	"github.com/nats-io/nats.go"

	"github.com/arnokay/arnobot-auth/internal/app/service"
)

type ProviderController struct {
	logger *slog.Logger

	providerService *service.AuthProviderService
}

func NewProviderController(
	providerService *service.AuthProviderService,
) *ProviderController {
	logger := applog.NewServiceLogger("mb-provider-controller")

	return &ProviderController{
		providerService: providerService,

		logger: logger,
	}
}

func (c *ProviderController) Connect(conn *nats.Conn) {
	conn.QueueSubscribe(topics.AuthProviderTokenGet, topics.AuthProviderTokenGet, c.get)
	conn.QueueSubscribe(topics.AuthProviderTokenUpdateTokens, topics.AuthProviderTokenUpdateTokens, c.updateTokens)
}

func (c *ProviderController) get(msg *nats.Msg) {
	var req apptype.Request[data.AuthProviderGet]
	var res apptype.Response[*data.AuthProvider]

	err := req.Decode(msg.Data)
	if err != nil {
		res.ToFailErr(err)
		b, _ := res.Encode()
		msg.Respond(b)
		return
	}

	ctx, cancel := newControllerContext(req.TraceID)
	defer cancel()
  res.TraceID = req.TraceID

	provider, err := c.providerService.Get(ctx, req.Data)
	if err != nil {
		res.ToFailErr(err)
		b, _ := res.Encode()
		msg.Respond(b)
		return
	}

	res.ToSuccess(provider)
	b, _ := res.Encode()
	msg.Respond(b)
}

func (c *ProviderController) updateTokens(msg *nats.Msg) {
	var req apptype.Request[data.AuthProviderUpdateTokens]
	var res apptype.EmptyResponse

	err := req.Decode(msg.Data)
	if err != nil {
		res.ToFailErr(err)
		b, _ := res.Encode()
		msg.Respond(b)
		return
	}

	ctx, cancel := newControllerContext(req.TraceID)
	defer cancel()
  res.TraceID = req.TraceID

	err = c.providerService.UpdateTokens(ctx, req.Data)
	if err != nil {
		res.ToFailErr(err)
		b, _ := res.Encode()
		msg.Respond(b)
		return
	}

	res.ToSuccess(true)
	b, _ := res.Encode()
	msg.Respond(b)
}
