package middleware

import (
	"log/slog"

	"github.com/arnokay/arnobot-shared/applog"
)

type Middlewares struct {
  logger *slog.Logger
}

func New() *Middlewares {
  logger := applog.NewServiceLogger("app-middleware")

  return &Middlewares{
    logger: logger,
  }
}
