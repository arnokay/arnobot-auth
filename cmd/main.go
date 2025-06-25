package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/arnokay/arnobot-shared/applog"
	echoControllers "github.com/arnokay/arnobot-shared/controllers/echo"
	mbControllers "github.com/arnokay/arnobot-shared/controllers/mb"
	"github.com/arnokay/arnobot-shared/db"
	"github.com/arnokay/arnobot-shared/pkg/assert"
	sharedService "github.com/arnokay/arnobot-shared/service"
	"github.com/arnokay/arnobot-shared/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/arnokay/arnobot-auth/internal/api"
	"github.com/arnokay/arnobot-auth/internal/api/middleware"
	"github.com/arnokay/arnobot-auth/internal/app/config"
	"github.com/arnokay/arnobot-auth/internal/app/service"
	"github.com/arnokay/arnobot-auth/internal/mb"
)

const APP_NAME = "auth"

type application struct {
	logger *slog.Logger

	db        *pgxpool.Pool
	queries   db.Querier
	cache     jetstream.KeyValue
	msgBroker *nats.Conn
	api       *echo.Echo

	storage        storage.Storager
	services       *service.Services
	apiControllers echoControllers.Controller
	apiMiddlewares *middleware.Middlewares
	mbControllers  mbControllers.NatsController
}

func main() {
	var app application

	// load config
	cfg := config.Load()

	// load logger
	logger := applog.Init(APP_NAME, os.Stdout, cfg.Global.LogLevel)
	app.logger = logger

  logger.Debug(
    "config", 
    "whitelist", cfg.WhitelistEnabled,
  )

	// load db
	dbConn := openDB()
	app.db = dbConn

	// load message broker
	mbConn, _, kvstore := openMB()
	app.msgBroker = mbConn
	app.cache = kvstore

	// load storage
	store := storage.NewStorage(app.db)
	app.storage = store

	// load services
	services := &service.Services{}
  services.PlatformAPIService = service.NewPlatformAPIService()
	services.ProviderService = service.NewAuthProviderService(app.storage)
	services.UserService = service.NewUserService(app.storage)
	services.SessionService = service.NewSessionService(app.storage)
	services.TransactionService = sharedService.NewPgxTransactionService(app.db)
	services.WhitelistService = service.NewWhitelistService(app.storage)
	services.OAuthService = service.NewOAuthService(app.cache)
	app.services = services

	// load middlewares
	app.apiMiddlewares = middleware.New(
		middleware.NewAuthMiddlewares(app.services.SessionService),
	)

	// load api controllers
	app.apiControllers = &api.Controllers{
		ProviderController: api.NewProviderController(
			app.services.PlatformAPIService,
			app.services.UserService,
			app.services.ProviderService,
			app.services.SessionService,
			app.services.TransactionService,
			app.services.WhitelistService,
			app.services.OAuthService,
		),
	}

	// load message broker controllers
	app.mbControllers = &mb.Controllers{
		SessionController:  mb.NewSessionController(app.services.SessionService),
		ProviderController: mb.NewProviderController(app.services.ProviderService),
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

func openMB() (*nats.Conn, jetstream.JetStream, jetstream.KeyValue) {
	nc, err := nats.Connect(config.Config.MB.URL)
	assert.NoError(err, "openMB: cannot open message broker connection")

	js, err := jetstream.New(nc)
	assert.NoError(err, "openMB: cannot create jetstream")
	kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket: "default-auth",
	})
	assert.NoError(err, "openMB: cannot create default kv store")

	return nc, js, kv
}
