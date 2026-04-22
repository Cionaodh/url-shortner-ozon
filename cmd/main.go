package main

import (
	"log/slog"
	"os"

	"github.com/Cionaodh/url-shortner-ozon/internal/app"
	"github.com/Cionaodh/url-shortner-ozon/internal/config"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// Config
	cfg, err := config.NewConfig()
	if err != nil {
		log.Error("config error", slog.Any("error", err))
		os.Exit(1)
	}

	// app.Migrations(log)
	app.Run(cfg, log)
}
