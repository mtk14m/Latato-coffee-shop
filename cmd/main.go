package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mtk14m/latato/internal/config"
	"github.com/mtk14m/latato/internal/handler"
	"github.com/mtk14m/latato/internal/repository"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.LoadEnv()
	if err != nil {
		logger.Error("chargement de la config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := repository.NewPostgresPool(ctx, cfg.DB.DSN())
	if err != nil {
		logger.Error("connexion à la base de données", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	router := handler.NewRouter(pool)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("serveur démarré", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("le serveur a planté", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("signal d'arrêt reçu, shutdown en cours...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown forcé (timeout dépassé)", "err", err)
	} else {
		logger.Info("serveur arrêté proprement")
	}
}
