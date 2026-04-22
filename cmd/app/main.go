package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "github.com/Black1black/go_base_api/configs"
	"github.com/Black1black/go_base_api/internal/db"
	"github.com/Black1black/go_base_api/internal/handler"
	walletRepo "github.com/Black1black/go_base_api/internal/repository/wallet"
	walletUC "github.com/Black1black/go_base_api/internal/usecase/wallet"
	"github.com/Black1black/go_base_api/pkg/logger"
)

// @title GoBaseApi API
// @version 1.0
// @description This is a sample API with Gin, GORM, and Worker Pool.

// @host localhost:8080
// @BasePath /api/v1

func main() {
	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	appLogger := logger.New(cfg.App.LoggingLevel, 1000)
	defer appLogger.Sync() // Важно для graceful shutdown

	appLogger.Info("go_base_api init try")

	// Initialize database
	postgresDB, err := db.InitPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.CloseConnection(postgresDB)

	walletRepo := walletRepo.NewRepository(postgresDB)
	defer walletRepo.Shutdown()

	walletUseCase := walletUC.NewUsecase(walletRepo)

	handler := handler.NewHandler(walletUseCase)

	// Initialize router
	router := handler.InitRoutes()

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	appLogger.Info("Server starting", "port", cfg.App.Port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", err)
	}

	appLogger.Info("Server stopped")
}
