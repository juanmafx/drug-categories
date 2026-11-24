package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/juanmafx/drug-categories/backend/internal/config"
	"github.com/juanmafx/drug-categories/backend/internal/graphqlapi"
	"github.com/juanmafx/drug-categories/backend/internal/httpapi"
	"github.com/juanmafx/drug-categories/backend/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repository, closeRepository, err := openRepository(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeRepository()

	service, err := graphqlapi.NewService(repository, logger)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr: cfg.Address,
		Handler: httpapi.NewHandler(service, httpapi.Options{
			AllowedOrigins: cfg.AllowedOrigins,
			WriteToken:     cfg.WriteToken,
			Logger:         logger,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	failures := make(chan error, 1)
	go func() {
		logger.Info("backend listening", "address", cfg.Address, "durable", cfg.DatabasePath != "")
		failures <- server.ListenAndServe()
	}()

	select {
	case err := <-failures:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownContext)
}

func openRepository(ctx context.Context, cfg config.Config) (store.Repository, func(), error) {
	if cfg.DatabasePath == "" {
		return store.NewMemory(store.DefaultSeed()...), func() {}, nil
	}
	sqliteStore, err := store.OpenSQLite(ctx, cfg.DatabasePath)
	if err != nil {
		return nil, nil, err
	}
	return sqliteStore, func() { _ = sqliteStore.Close() }, nil
}
