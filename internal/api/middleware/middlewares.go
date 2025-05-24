package middleware

import (
	"log/slog"

	"arnobot-shared/applog"
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
