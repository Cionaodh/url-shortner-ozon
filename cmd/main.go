package main

import (
	"fmt"
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
		log.Info(fmt.Sprintf("Config error: %s", err))
		os.Exit(1)
	}

	// app.Migrations(log)
	app.Run(cfg, log)
}
