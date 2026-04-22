package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	config "github.com/Black1black/go_base_api/configs"
	"github.com/Black1black/go_base_api/internal/db"
	"github.com/Black1black/go_base_api/internal/handler"
	walletUC "github.com/Black1black/go_base_api/internal/usecase/wallet"

	"github.com/Black1black/go_base_api/internal/repository/wallet"
	"github.com/Black1black/go_base_api/pkg/logger"

	"log"
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

	walletRepo := wallet.NewRepository(postgresDB)

	walletUseCase := walletUC.NewUsecase(walletRepo)

	handler := handler.NewHandler(walletUseCase)

	// Initialize router
	router := handler.InitRoutes()

	// Start server
	appLogger.Info("Server starting", "port", cfg.App.Port)
	go func() {
		if err := router.Run(fmt.Sprintf(":%d", cfg.App.Port)); err != nil {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

}
