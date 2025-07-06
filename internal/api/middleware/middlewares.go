package middleware

import (
	

	"github.com/arnokay/arnobot-shared/applog"
)

type Middlewares struct {
	AuthMiddlewares *AuthMiddlewares

	logger applog.Logger
}

func New(authMiddewares *AuthMiddlewares) *Middlewares {
	logger := applog.NewServiceLogger("app-middleware")

	return &Middlewares{
		AuthMiddlewares: authMiddewares,

		logger: logger,
	}
}
