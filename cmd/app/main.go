package main

import (
	config "ex_proj_go/configs"
	"ex_proj_go/internal/db"
	"ex_proj_go/internal/handler"

	"ex_proj_go/internal/repository/users"
	"go_base_api/pkg/logger"

	"log"
)

// @title ExProjectGo API
// @version 1.0
// @description This is a sample API with Gin, GORM, Swagger, and Worker Pool.

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

	appLogger.Info("go_base_api init try",

		"check_time", cfg.App.CheckTime,
	)

	// Initialize database
	postgresDB, err := db.InitPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.CloseConnection(postgresDB)

	usersRepo := users.NewRepository(postgresDB)

	usersUseCase := usersUC.NewUsecase(usersRepo)

	handler := handler.NewHandler(usersUseCase)

}
