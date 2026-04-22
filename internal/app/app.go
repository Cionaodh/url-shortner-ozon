package app

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Cionaodh/url-shortner-ozon/internal/config"
	"github.com/Cionaodh/url-shortner-ozon/internal/controller/http"
	"github.com/Cionaodh/url-shortner-ozon/internal/repo/pgdb"
	"github.com/Cionaodh/url-shortner-ozon/internal/service"
	"github.com/Cionaodh/url-shortner-ozon/pkg/httpserver"
	"github.com/Cionaodh/url-shortner-ozon/pkg/postgres"
)

func Run(cfg *config.Config, log *slog.Logger) {

	pg, err := postgres.New(cfg.PG.Conn, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		log.Error("failed to init postgres", slog.Any("error", err))
		return
	}
	defer pg.Close()

	s := service.NewShortnerService(pgdb.NewStorage(pg), log)

	router := http.NewRouter(s, log)

	httpServer := httpserver.New(router, httpserver.Port(cfg.HTTP.Port))
	httpServer.Start()

	log.Info("server started", slog.String("port", cfg.HTTP.Port))

	// Graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		log.Info("stopping the app on signal", slog.String("signal", s.String()))
	case err := <-httpServer.Notify():
		log.Error("stopping the app on server error", slog.Any("error", err))
	}

	log.Info("starting graceful shutdown")

	// Выполняем graceful shutdown
	if err := httpServer.Shutdown(); err != nil {
		log.Error("error when stopping the server", slog.Any("error", err))
	} else {
		log.Info("server successfully stopped")
	}
}
