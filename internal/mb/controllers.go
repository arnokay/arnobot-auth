package mb

import (
	"context"
	"time"

	"arnobot-shared/controllers/mb"
	"arnobot-shared/trace"
)

type Controllers struct {
	SessionController controllers.Controller
}

func (c *Controllers) Connect() {
	c.SessionController.Connect()
}

func newControllerContext(traceID string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ctx = trace.Context(ctx, traceID)

	return ctx, cancel
}
