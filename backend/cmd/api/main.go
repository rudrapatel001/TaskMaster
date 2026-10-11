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

	"github.com/rudrapatel001/taskmaster/backend/internal/auth"
	"github.com/rudrapatel001/taskmaster/backend/internal/config"
	"github.com/rudrapatel001/taskmaster/backend/internal/database"
	"github.com/rudrapatel001/taskmaster/backend/internal/handlers"
	"github.com/rudrapatel001/taskmaster/backend/internal/middleware"
	"github.com/rudrapatel001/taskmaster/backend/internal/repository"
	"github.com/rudrapatel001/taskmaster/backend/internal/router"
	"github.com/rudrapatel001/taskmaster/backend/internal/services"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg.Env)

	// ctx is cancelled when the process receives Ctrl+C or SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	logger.Info("database connected")

	// Build the layers from the bottom up: repository -> service -> handler.
	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(pool)
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, 15*time.Minute)
	authService := services.NewAuthService(userRepo, refreshTokenRepo, tokenManager)
	userService := services.NewUserService(userRepo)
	workspaceRepo := repository.NewWorkspaceRepository(pool)
	workspaceService := services.NewWorkspaceService(workspaceRepo)

	handler := router.New(router.Deps{
		Logger:         logger,
		AllowedOrigins: cfg.CORSOrigins,
		Health:         &handlers.HealthHandler{DB: pool, Logger: logger},
		Auth:           handlers.NewAuthHandler(authService, logger, cfg.SecureCookies),
		Users:          handlers.NewUserHandler(userService, logger),
		Workspaces:     handlers.NewWorkspaceHandler(workspaceService, logger),
		Authenticate:   middleware.Authenticate(tokenManager),
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down")
	}

	// Give in-flight requests up to 10 seconds to finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Info("server stopped")
	return nil
}

func newLogger(env string) *slog.Logger {
	if env == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}
