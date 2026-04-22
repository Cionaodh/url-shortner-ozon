package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type (
	Config struct {
		HTTP HTTP
		PG   PG
	}

	HTTP struct {
		Port string `env:"HTTP_PORT,required"`
	}

	PG struct {
		PoolMax int    `env:"PG_POOL_MAX" envDefault:"10"`
		Conn    string `env:"PG_CONN,required"` // обязательное поле
	}
)

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	return cfg, nil
}
