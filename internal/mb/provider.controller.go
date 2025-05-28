package mb

import (
	"log/slog"

	"arnobot-shared/applog"
	"arnobot-shared/mbtypes"
	"arnobot-shared/topics"
	"github.com/nats-io/nats.go"

	"arnobot-auth/internal/app/service"
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
	var req mbtypes.AuthProviderGetRequest
	var res mbtypes.AuthProviderGetResponse

	err := req.Decode(msg.Data)
	if err != nil {
		res.ToFailErr(err)
		b, _ := res.Encode()
		msg.Respond(b)
		return
	}

	ctx, cancel := newControllerContext(req.TraceID)
	defer cancel()

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
	var req mbtypes.AuthProviderUpdateTokensRequest
	var res mbtypes.AuthProviderUpdateTokensResponse

	err := req.Decode(msg.Data)
	if err != nil {
		res.ToFailErr(err)
		b, _ := res.Encode()
		msg.Respond(b)
		return
	}

	ctx, cancel := newControllerContext(req.TraceID)
	defer cancel()

	err = c.providerService.UpdateTokens(ctx, req.Data.ID, req.Data.Data)
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
