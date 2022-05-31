package config

import (
	"flag"
	"net/url"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/rs/zerolog"
)

// напрашивается переделать на hostname + port, потому что сейчас нужно
// внимательно управлять тремя параметрами в связке - ServerAddr, BaseAddr,
// CookieDomain. с другой стороны, надо помнить, что хост и порт который
// мы слушаем это одно, а как мы доступны снаружи - возможно, совсем другое
const (
	DefaultServerAddr   = "localhost:8080"
	DefaultLogName      = "app.log"
	DefaultBaseAddr     = "http://localhost:8080/"
	DefaultCookieDomain = "localhost"
)

type Config struct {
	ServerAddr      string `env:"SERVER_ADDRESS"`
	LogName         string `env:"LOG_NAME"`
	LogFile         *os.File
	Logger          *zerolog.Logger
	BaseAddr        string `env:"BASE_URL"`
	BaseURI         *url.URL
	StorageFileName string `env:"FILE_STORAGE_PATH"`
	CookieDomain    string `env:"COOKIE_DOMAIN"`
	// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
	DBConnectString string `env:"DATABASE_DSN"`
}

func ParseConfig() *Config {
	var config Config
	err := env.Parse(&config)
	if err != nil {
		panic(err)
	}

	var flagServerAddr, flagBaseAddr, flagCookieDomain, flagStorageFileName, flagDBConnectString string
	flag.StringVar(&flagServerAddr, "a", DefaultServerAddr, "host:port to listen on")
	flag.StringVar(&flagBaseAddr, "b", DefaultBaseAddr, "base addr for shortened urls")
	flag.StringVar(&flagCookieDomain, "cd", DefaultCookieDomain, "cookie domain")
	flag.StringVar(&flagStorageFileName, "f", "", "filepath to store shortened urls")
	flag.StringVar(&flagDBConnectString, "d", "", "db connect string")
	flag.Parse()

	// приоритет ENV перед флагами сконструирован руками, в каждом из присвоений
	// итоговых полей конфига. не очень хорошо. обёртку бы написать
	if len(config.ServerAddr) == 0 {
		if len(flagServerAddr) > 0 {
			config.ServerAddr = flagServerAddr
		} else {
			config.ServerAddr = DefaultServerAddr
		}
	}

	if len(config.LogName) == 0 {
		config.LogName = DefaultLogName
	}
	config.LogFile, err = os.OpenFile(config.LogName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	// TODO: сделать так, чтобы zerolog всегда писал в файл; не только когда
	// TODO: мы берём этот объект, но и просто при вызове log где угодно
	logger := zerolog.New(config.LogFile).With().Timestamp().Logger()
	config.Logger = &logger

	if len(config.BaseAddr) == 0 {
		if len(flagBaseAddr) > 0 {
			config.BaseAddr = flagBaseAddr
		} else {
			config.BaseAddr = DefaultBaseAddr
		}
	}
	config.BaseURI, err = url.Parse(config.BaseAddr)
	if err != nil {
		panic(err)
	}

	if len(config.StorageFileName) == 0 {
		if len(flagStorageFileName) > 0 {
			config.StorageFileName = flagStorageFileName
		}
	}

	if len(config.CookieDomain) == 0 {
		if len(flagCookieDomain) > 0 {
			config.CookieDomain = flagCookieDomain
		} else {
			config.CookieDomain = DefaultCookieDomain
		}
	}

	if len(config.DBConnectString) == 0 {
		if len(flagDBConnectString) > 0 {
			config.DBConnectString = flagDBConnectString
		}
	}

	return &config
}
