package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
)

// напрашивается переделать на hostname + port, потому что сейчас нужно
// внимательно управлять тремя параметрами в связке - ServerAddr, BaseAddr,
// CookieDomain. с другой стороны, надо помнить, что хост и порт который
// мы слушаем это одно, а как мы доступны снаружи - возможно, совсем другое
const (
	DefaultServerAddr   = "localhost:8080"
	DefaultLogName      = "app.log" // путь от корневой папки репозитория
	DefaultBaseAddr     = "http://localhost:8080/"
	DefaultCookieDomain = "localhost"
)

type Config struct {
	ServerAddr      string `env:"SERVER_ADDRESS"`
	LogName         string `env:"LOG_NAME"`
	BaseAddr        string `env:"BASE_URL"`
	StorageFileName string `env:"FILE_STORAGE_PATH"`
	CookieDomain    string `env:"COOKIE_DOMAIN"`
	// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
	DBConnectString  string `env:"DATABASE_DSN"`
	DBConnectTimeout int    `env:"DATABASE_CONNECT_TIMEOUT"` // в секундах, дефолт - 1
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

	if len(config.BaseAddr) == 0 {
		if len(flagBaseAddr) > 0 {
			config.BaseAddr = flagBaseAddr
		} else {
			config.BaseAddr = DefaultBaseAddr
		}
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
	// вообще-то для постгреса значение "0" означает, что таймаут отключен;
	// но нам как-то надо же проставлять дефолтное значение, которое 1 (делать
	// дефолтом 0 было бы нехорошо, дефолты должны быть разумны).
	// если правда нужно отключить таймаут - то отрицательное значение подойдёт
	if config.DBConnectTimeout == 0 {
		config.DBConnectTimeout = 1
	}

	return &config
}
