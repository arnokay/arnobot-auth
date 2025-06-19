package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/arnokay/arnobot-shared/middlewares"
	"github.com/labstack/echo/v4"

	"arnobot-auth/internal/app/config"
)

func (app *application) Start() {
	startError := make(chan error)
	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		s := <-quit

		app.logger.Info("shutting down server", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		shutdownError <- app.Shutdown(ctx)
	}()

	go func() {
		err := startAPIServer(app)
		if err != nil {
			startError <- err
		}
	}()

	go func() {
		err := startMBServer(app)
		if err != nil {
			startError <- err
		}
	}()
	select {
	case err := <-startError:
		app.logger.Error("application start error", "err", err)
	case err := <-shutdownError:
		if err != nil {
			app.logger.Error("application shutdown error", "err", err)
		}
	}
	close(startError)
	close(shutdownError)
}

func (app *application) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		app.logger.Debug("#shutdown.mb: gracefully closing mb connection")

		done := make(chan struct{})
		go func() {
			err := app.msgBroker.Drain()
			if err != nil {
				errCh <- err
			}
			close(done)
		}()

		select {
		case <-done:
			app.logger.Debug("#shutdown.mb: gracefully closed mb connection")
		case <-ctx.Done():
			app.logger.Debug("#shutdown.mb: context timeout, force closing message broker connection")
			app.msgBroker.Close()
			app.logger.Debug("#shutdown:mb: force closed mb connection")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		app.logger.Debug("#shutdown.api: gracefully closing api")
		err := app.api.Shutdown(ctx)
		if err != nil {
			errCh <- err
			return
		}
		app.logger.Debug("#shutdown.api: gracefully closed api")
	}()

	wg.Wait()
	close(errCh)

	app.logger.Debug("#shutdown.db: gracefully closing db")
	app.db.Close()
	app.logger.Debug("#shutdown.db: gracefully closed db")

	for err := range errCh {
		return err
	}

	return nil
}

func startMBServer(a *application) error {
	if a.msgBroker == nil {
		return errors.New("startMBServer: msgBroker is nil")
	}

	if a.msgBroker.IsClosed() {
		return errors.New("startMBServer: msgBroker is closed")
	}

	a.mbControllers.Connect(a.msgBroker)

	return nil
}

func startAPIServer(a *application) error {
	e := echo.New()

	e.HideBanner = true
	a.api = e

	e.Use(middlewares.AttachTraceID)

	mainGroup := e.Group("/v1")
	a.apiControllers.Routes(mainGroup)

	e.HTTPErrorHandler = middlewares.ErrHandler

	err := e.Start(fmt.Sprintf(":%v", config.Config.Global.Port))
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
