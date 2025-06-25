package middleware

import (
	"log/slog"

	"github.com/arnokay/arnobot-shared/applog"
)

type Middlewares struct {
	AuthMiddlewares *AuthMiddlewares

	logger *slog.Logger
}

func New(authMiddewares *AuthMiddlewares) *Middlewares {
	logger := applog.NewServiceLogger("app-middleware")

	return &Middlewares{
		AuthMiddlewares: authMiddewares,

		logger: logger,
	}
}
