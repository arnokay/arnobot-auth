package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/arnokay/arnobot-shared/pkg/assert"
	"github.com/arnokay/arnobot-shared/platform"
)

const (
	ENV_PORT                 = "PORT"
	ENV_DB_DSN               = "DB_DSN"
	ENV_TWITCH_CLIENT_ID     = "TWITCH_CLIENT_ID"
	ENV_TWITCH_CLIENT_SECRET = "TWITCH_CLIENT_SECRET"
	ENV_TWITCH_REDIRECT_URI  = "TWITCH_REDIRECT_URI"
	ENV_FRONT_END_CALLBACK   = "FRONT_END_CALLBACK"
	ENV_MB_URL               = "MB_URL"
	ENV_WHITELIST_ENABLED    = "WHITELIST_ENABLED"
)

type config struct {
	Global           GlobalConfig
	DB               DBConfig
	MB               MBConfig
	Providers        map[platform.Platform]*ProviderConfig
	FrontEndCallback string
	WhitelistEnabled bool
}

type GlobalConfig struct {
	Env      string
	Port     int
	LogLevel int
}

type MBConfig struct {
	URL string
}

type DBConfig struct {
	DSN          string
	MaxIdleConns int
	MaxOpenConns int
	MaxIdleTime  string
}

type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

var Config *config

func Load() *config {
	Config = &config{
		Global: GlobalConfig{
			Port:     3000,
			LogLevel: -4,
		},
		Providers: make(map[platform.Platform]*ProviderConfig),
	}

	twitchProvider := ProviderConfig{
		ClientID:     os.Getenv(ENV_TWITCH_CLIENT_ID),
		ClientSecret: os.Getenv(ENV_TWITCH_CLIENT_SECRET),
		RedirectURI:  os.Getenv(ENV_TWITCH_REDIRECT_URI),
	}

	Config.Providers[platform.Twitch] = &twitchProvider

	if os.Getenv(ENV_PORT) != "" {
		port, err := strconv.Atoi(os.Getenv(ENV_PORT))
		assert.NoError(err, fmt.Sprintf("%v: not a number", ENV_PORT))
		Config.Global.Port = port
	}

	flag.StringVar(&Config.Global.Env, "env", "development", "Environment (development|staging|production)")

  flag.BoolVar(&Config.WhitelistEnabled, "whitelist", false, "enable whitelist")
	flag.IntVar(&Config.Global.Port, "port", Config.Global.Port, "Server Port")
	flag.IntVar(&Config.Global.LogLevel, "log-level", Config.Global.LogLevel, "Minimal Log Level (default: -4)")

	flag.StringVar(&Config.DB.DSN, "db-dsn", os.Getenv(ENV_DB_DSN), "DB DSN")
	flag.IntVar(&Config.DB.MaxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.IntVar(&Config.DB.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.StringVar(&Config.DB.MaxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")
	flag.StringVar(&Config.FrontEndCallback, "fe-callback", os.Getenv(ENV_FRONT_END_CALLBACK), "where to redirect after creating session")
	flag.StringVar(&Config.MB.URL, "mb-url", os.Getenv(ENV_MB_URL), "Message Broker URL")

	flag.StringVar(
		&twitchProvider.ClientID,
		"provider-twitch-client-id",
		os.Getenv(ENV_TWITCH_CLIENT_ID),
		"Twitch Provider Client ID",
	)
	flag.StringVar(
		&twitchProvider.ClientSecret,
		"privder-twitch-client-secret",
		os.Getenv(ENV_TWITCH_CLIENT_SECRET),
		"Twitch Provider Client Secret",
	)
	flag.StringVar(
		&twitchProvider.RedirectURI,
		"provider-twitch-redirect-uri",
		os.Getenv(ENV_TWITCH_REDIRECT_URI),
		"Twitch Provider Redirect URI",
	)

	flag.Parse()

	return Config
}
