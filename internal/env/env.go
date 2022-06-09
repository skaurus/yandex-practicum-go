package env

import (
	"context"
	"net/url"
	"os"
	"time"

	"github.com/skaurus/yandex-practicum-go/internal/config"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/rs/zerolog"
)

type Environment struct {
	Config  *config.Config
	LogFile *os.File
	Logger  *zerolog.Logger
	BaseURI *url.URL
	DBConn  *pgx.Conn
}

func New() (*Environment, error) {
	var err error
	env := &Environment{
		Config: config.ParseConfig(),
	}

	env.BaseURI, err = url.Parse(env.Config.BaseAddr)
	if err != nil {
		panic(err)
	}

	env.LogFile, err = os.OpenFile(env.Config.LogName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	// TODO: сделать так, чтобы zerolog всегда писал в файл; не только когда
	// TODO: мы берём этот объект, но и просто при вызове log где угодно
	logger := zerolog.New(env.LogFile).With().Timestamp().Logger()
	env.Logger = &logger

	if len(env.Config.DBConnectString) > 0 {
		connConfig, err := pgx.ParseConfig(env.Config.DBConnectString)
		if err != nil {
			return nil, err
		}
		connConfig.Logger = zerologadapter.NewLogger(*env.Logger)
		//connConfig.LogLevel = pgx.LogLevelError // можно использовать для дебага

		ctx, cancel := context.WithTimeout(
			context.Background(),
			time.Second*time.Duration(env.Config.DBConnectTimeout),
		)
		defer cancel()
		env.DBConn, err = pgx.ConnectConfig(ctx, connConfig)
		if err != nil {
			return nil, err
		}
		cancel()
	}

	return env, nil
}
