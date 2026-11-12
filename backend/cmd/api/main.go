package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
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
	invitationRepo := repository.NewInvitationRepository(pool)
	invitationService := services.NewInvitationService(invitationRepo)
	projectRepo := repository.NewProjectRepository(pool)
	projectService := services.NewProjectService(projectRepo)
	taskRepo := repository.NewTaskRepository(pool)
	taskService := services.NewTaskService(taskRepo)
	commentRepo := repository.NewCommentRepository(pool)
	commentService := services.NewCommentService(commentRepo)
	activityRepo := repository.NewActivityRepository(pool)
	activityService := services.NewActivityService(activityRepo, logger)
	notificationRepo := repository.NewNotificationRepository(pool)
	notificationService := services.NewNotificationService(notificationRepo, logger)
	// One task event feeds both the activity feed and notifications.
	taskEvents := services.NewTaskEvents(activityService, notificationService)
	statusRepo := repository.NewStatusRepository(pool)
	statusService := services.NewStatusService(statusRepo)
	searchRepo := repository.NewSearchRepository(pool)
	searchService := services.NewSearchService(searchRepo)

	handler := router.New(router.Deps{
		Logger:           logger,
		AllowedOrigins:   cfg.CORSOrigins,
		Health:           &handlers.HealthHandler{DB: pool, Logger: logger},
		Auth:             handlers.NewAuthHandler(authService, logger, cfg.SecureCookies),
		Users:            handlers.NewUserHandler(userService, logger),
		Workspaces:       handlers.NewWorkspaceHandler(workspaceService, logger),
		Invitations:      handlers.NewInvitationHandler(invitationService, logger),
		Projects:         handlers.NewProjectHandler(projectService, logger),
		Tasks:            handlers.NewTaskHandler(taskService, logger),
		Authenticate:     middleware.Authenticate(tokenManager),
		WorkspaceAccess:  middleware.WorkspaceAccess(workspaceService, logger),
		Comments:         handlers.NewCommentHandler(commentService, logger),
		Activity:         handlers.NewActivityHandler(activityService, logger),
		ActivityRecorder: taskEvents,
		Notifications:    handlers.NewNotificationHandler(notificationService, logger),
		Statuses:         handlers.NewStatusHandler(statusService, logger),
		Search:           handlers.NewSearchHandler(searchService, logger),
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Claim the port first, so a problem (such as the port already being in
	// use) fails right here, before we announce anything.
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", srv.Addr, err)
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
