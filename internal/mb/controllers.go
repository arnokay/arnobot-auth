package mb

import (
	"context"
	"time"

	"arnobot-shared/controllers/mb"
	"arnobot-shared/trace"
	"github.com/nats-io/nats.go"
)

type Controllers struct {
	SessionController  controllers.NatsController
	ProviderController controllers.NatsController
}

func (c *Controllers) Connect(conn *nats.Conn) {
	c.SessionController.Connect(conn)
}

func newControllerContext(traceID string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ctx = trace.Context(ctx, traceID)

	return ctx, cancel
}
