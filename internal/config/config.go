package config

import (
	"flag"
	"net/url"

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
	LogLevel        zerolog.Level
	BaseAddr        string `env:"BASE_URL"`
	BaseURI         *url.URL
	StorageFileName string `env:"FILE_STORAGE_PATH"`
	CookieDomain    string `env:"COOKIE_DOMAIN"`
}

func ParseConfig() *Config {
	var config Config
	err := env.Parse(&config)
	if err != nil {
		panic(err)
	}

	var flagServerAddr, flagBaseAddr, flagCookieDomain, flagStorageFileName string
	flag.StringVar(&flagServerAddr, "a", DefaultServerAddr, "host:port to listen on")
	flag.StringVar(&flagBaseAddr, "b", DefaultBaseAddr, "base addr for shortened urls")
	flag.StringVar(&flagCookieDomain, "d", DefaultCookieDomain, "cookie domain")
	flag.StringVar(&flagStorageFileName, "f", "", "filepath to store shortened urls")
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
	config.LogLevel = zerolog.ErrorLevel

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

	return &config
}
