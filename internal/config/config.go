package config

import (
	"flag"
	"net/url"

	"github.com/caarlos0/env/v6"
)

const (
	DefaultServerAddr = "localhost:8080"
	DefaultLogName    = "app.log"
	DefaultBaseAddr   = "http://localhost:8080/"
)

type Config struct {
	ServerAddr      string `env:"SERVER_ADDRESS"`
	LogName         string `env:"LOG_NAME"`
	BaseAddr        string `env:"BASE_URL"`
	BaseURI         *url.URL
	StorageFileName string `env:"FILE_STORAGE_PATH"`
}

func ParseConfig() *Config {
	var config Config
	err := env.Parse(&config)
	if err != nil {
		panic(err)
	}

	var flagServerAddr, flagBaseAddr, flagStorageFileName string
	flag.StringVar(&flagServerAddr, "a", DefaultServerAddr, "host:port to listen on")
	flag.StringVar(&flagBaseAddr, "b", DefaultBaseAddr, "base addr for shortened urls")
	flag.StringVar(&flagStorageFileName, "f", "", "filepath to store shortened urls")
	flag.Parse()

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
	config.BaseURI, err = url.Parse(config.BaseAddr)
	if err != nil {
		panic(err)
	}
	if len(config.StorageFileName) == 0 {
		if len(flagStorageFileName) > 0 {
			config.StorageFileName = flagStorageFileName
		}
	}

	return &config
}
