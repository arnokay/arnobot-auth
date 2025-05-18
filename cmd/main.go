package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"arnobot-shared/applog"
	echoControllers "arnobot-shared/controllers/echo"
	mbControllers "arnobot-shared/controllers/mb"
	"arnobot-shared/db"
	"arnobot-shared/pkg"
	"arnobot-shared/pkg/assert"
	"arnobot-shared/pkg/mapcacher"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"

	"arnobot-auth/internal/api"
	"arnobot-auth/internal/api/middleware"
	"arnobot-auth/internal/app/config"
	"arnobot-auth/internal/app/service"
	"arnobot-auth/internal/mb"
)

const APP_NAME = "auth"

type application struct {
	logger *slog.Logger

	db        *pgxpool.Pool
	queries   db.Querier
	store     pkg.Cacher
	msgBroker *nats.Conn
	api       *echo.Echo

	services       *service.Services
	apiControllers echoControllers.Controller
	apiMiddlewares *middleware.Middlewares
	mbControllers  mbControllers.Controller
}

func main() {
	var app application

	// load config
	cfg := config.Load()

	// load logger
	logger := applog.Init(APP_NAME, os.Stdout, cfg.Global.LogLevel)
	app.logger = logger

	// load db
	dbConn := openDB()
	defer dbConn.Close()
	queries := db.New(dbConn)
	app.db = dbConn
	app.queries = queries

	// load cache
	store := mapcacher.New()
	app.store = &store

	// load message broker
	mbConn := openMB()
	app.msgBroker = mbConn

	// load services
	app.services = &service.Services{
		TwitchService:   service.NewTwitchApiService(app.store),
		ProviderService: service.NewAuthProviderService(app.queries),
		UserService:     service.NewUserService(app.queries),
		SessionService:  service.NewSessionService(app.queries),
	}

  // load middlewares
  app.apiMiddlewares = middleware.New()

	// load api controllers
	app.apiControllers = &api.Controllers{
		ProviderController: api.NewProviderController(
			app.services.TwitchService,
			app.services.UserService,
			app.services.ProviderService,
			app.services.SessionService,
		),
	}

	// load message broker controllers
	app.mbControllers = &mb.Controllers{
		SessionController: mb.NewSessionController(app.msgBroker, app.services.SessionService),
	}

	app.Start()
}

func openDB() *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(config.Config.DB.DSN)
	assert.NoError(err, "openDB: cannot open database connection")

	cfg.MaxConns = int32(config.Config.DB.MaxOpenConns)
	cfg.MinConns = int32(config.Config.DB.MaxIdleConns)

	duration, err := time.ParseDuration(config.Config.DB.MaxIdleTime)
	assert.NoError(err, "openDB: cannot parse duration")

	cfg.MaxConnIdleTime = duration

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	assert.NoError(err, "openDB: cannot open database connection")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pool.Ping(ctx)
	assert.NoError(err, "openDB: cannot ping")

	return pool
}

// func openDB(ctx context.Context) *sql.DB {
// 	db, err := sql.Open("postgres", config.Config.DB.DSN)
// 	assert.NoError(err, "openDB: cannot open database connection")
//
// 	db.SetMaxIdleConns(config.Config.DB.MaxIdleConns)
// 	db.SetMaxOpenConns(config.Config.DB.MaxOpenConns)
//
// 	duration, err := time.ParseDuration(config.Config.DB.MaxIdleTime)
// 	assert.NoError(err, "openDB: cannot parse duration")
//
// 	db.SetConnMaxIdleTime(duration)
//
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()
//
// 	err = db.PingContext(ctx)
// 	assert.NoError(err, "openDB: cannot ping")
//
// 	return db
// }

func openMB() *nats.Conn {
	nc, err := nats.Connect(config.Config.MB.URL)
	assert.NoError(err, "openMB: cannot open message broker connection")

	return nc
}
