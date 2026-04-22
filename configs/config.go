package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Postgres struct {
		Host     string
		Port     int
		User     string
		Password string
		DBName   string
	}
	App struct {
		Port         int
		LoggingLevel string
		getAPIURL    string
	}
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load("configs/.env"); err != nil {
		// Пробуем загрузить из корня, если нет в configs
		if err := godotenv.Load(); err != nil {
			return &Config{}, fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	var cfg Config

	cfg.Postgres.Host = os.Getenv("POSTGRES_HOST")
	cfg.Postgres.User = os.Getenv("POSTGRES_USER")
	cfg.Postgres.Password = os.Getenv("POSTGRES_PASSWORD")
	cfg.Postgres.DBName = os.Getenv("POSTGRES_DBNAME")

	if postgresPortStr := os.Getenv("POSTGRES_PORT"); postgresPortStr != "" {
		postgresPort, err := strconv.Atoi(postgresPortStr)
		if err != nil {
			return &Config{}, fmt.Errorf("invalid POSTGRES_PORT: %w", err)
		}
		cfg.Postgres.Port = postgresPort
	} else {
		cfg.Postgres.Port = 5432
	}

	cfg.App.LoggingLevel = os.Getenv("APP_LOGING_LEVEL")

	if appPortStr := os.Getenv("APP_PORT"); appPortStr != "" {
		appPort, err := strconv.Atoi(appPortStr)
		if err != nil {
			return &Config{}, fmt.Errorf("invalid APP_PORT: %w", err)
		}
		cfg.App.Port = appPort
	} else {
		cfg.App.Port = 8080
	}

	if err := validateConfig(&cfg); err != nil {
		return &Config{}, err
	}

	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	if cfg.Postgres.Host == "" {
		return fmt.Errorf("POSTGRES_HOST is required")
	}
	if cfg.Postgres.User == "" {
		return fmt.Errorf("POSTGRES_USER is required")
	}
	if cfg.Postgres.DBName == "" {
		return fmt.Errorf("POSTGRES_DBNAME is required")
	}

	if cfg.App.LoggingLevel == "" {
		return fmt.Errorf("APP_LOGING_LEVEL is required")
	}

	return nil
}
