package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpapi "pull-request-service/internal/http"
	"pull-request-service/internal/repository"
	"pull-request-service/internal/service"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable is required")
	}

	db, err := repository.NewPostgres(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to init postgres: %v", err)
	}
	defer db.Pool.Close()

	teamRepo := repository.NewTeamRepo(db)
	userRepo := repository.NewUserRepo(db)
	prRepo := repository.NewPRRepo(db)

	teamService := service.NewTeamService(teamRepo)
	userService := service.NewUserService(userRepo)
	prService := service.NewPRService(prRepo, userRepo)

	handler := httpapi.NewHandler(teamService, userService, prService, logger)

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler.Router(),
	}

	go func() {
		logger.Info("starting http server", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.Any("err", err))
			cancel()
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	logger.Info("shutting down server")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Error("server shutdown error", slog.Any("err", err))
	}

	logger.Info("server stopped")
}
