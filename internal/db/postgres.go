package db

import (
	config "ex_proj_go/configs"
	"ex_proj_go/internal/models"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitPostgresDB создает и возвращает новое подключение к БД
func InitPostgresDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		cfg.Postgres.Host,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBName,
		cfg.Postgres.Port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true, // Включаем подготовку выражений для повышения производительности
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// TODO - настроить пулл соединений
	// // Настройка пула соединений
	// sqlDB, err := db.DB()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	// }

	// sqlDB.SetMaxOpenConns(cfg.Postgres.MaxOpenConns)       // Максимальное число открытых соединений
	// sqlDB.SetMaxIdleConns(cfg.Postgres.MaxIdleConns)       // Максимальное число простаивающих соединений
	// sqlDB.SetConnMaxLifetime(cfg.Postgres.ConnMaxLifetime) // Максимальное время жизни соединения

	// Применение миграций
	if err := ApplyMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return db, nil
}

// CloseConnection безопасно закрывает соединение с БД
func CloseConnection(db *gorm.DB) {
	if db == nil {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Failed to get sql.DB from gorm.DB: %v", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("Failed to close database connection: %v", err)
	}
}

// ApplyMigrations применяет все необходимые миграции
func ApplyMigrations(db *gorm.DB) error {
	models := []interface{}{
		&models.User{},
		// Добавьте другие модели здесь
	}

	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate model %T: %w", model, err)
		}
		log.Printf("Successfully migrated model: %T", model)
	}

	return nil
}
