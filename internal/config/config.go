package config

import (
	"github.com/caarlos0/env/v6"
	"net/url"
)

type Config struct {
	ServerAddr string `env:"SERVER_ADDR"`
	LogName    string `env:"LOG_NAME"`
	BaseAddr   string `env:"BASE_ADDR"`
	BaseURI    *url.URL
}

func ParseConfig() *Config {
	var config Config
	err := env.Parse(&config)
	if err != nil {
		panic(err)
	}

	if len(config.ServerAddr) == 0 {
		config.ServerAddr = "localhost:8080"
	}
	if len(config.LogName) == 0 {
		config.LogName = "app.log"
	}
	if len(config.BaseAddr) == 0 {
		config.BaseAddr = "http://localhost:8080/"
	}
	config.BaseURI, err = url.Parse(config.BaseAddr)
	if err != nil {
		panic(err)
	}

	return &config
}
