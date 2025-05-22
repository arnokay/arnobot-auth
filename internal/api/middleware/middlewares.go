package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"arnobot-shared/applog"
	"arnobot-shared/pkg/errs"
	"arnobot-shared/trace"

	"github.com/labstack/echo/v4"
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

func (m *Middlewares) ErrHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	status := http.StatusInternalServerError
  responseErr := errs.ErrInternal

	var appErr *errs.AppError
	if errors.As(err, &appErr) {
		status = errs.ToHTTPStatus(appErr)
    responseErr = *appErr
	} else {
		if he, ok := err.(*echo.HTTPError); ok {
			status = he.Code
      responseErr = errs.New(errs.CodeHTTP, he.Error(), he)
		}
	}

	c.JSON(status, responseErr)
}

func (m *Middlewares) AttachTraceID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		traceID := trace.New()

		c.Set(trace.TraceIDKey.String(), traceID)
		ctx2 := context.WithValue(c.Request().Context(), trace.TraceIDKey, traceID)
		c.SetRequest(c.Request().WithContext(ctx2))
		c.Response().Header().Add("X-Trace-ID", traceID)

		return next(c)
	}
}
