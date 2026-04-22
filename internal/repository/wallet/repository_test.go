package wallet

import (
	"context"
	"fmt"
	"os"
	"testing"

	config "github.com/Black1black/go_base_api/configs"
	"github.com/Black1black/go_base_api/internal/models"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	dbHost := cfg.Postgres.Host
	if os.Getenv("IN_DOCKER") == "true" {
		dbHost = os.Getenv("TEST_DB_HOST")
		if dbHost == "" {
			dbHost = "test-db"
		}
	} else if os.Getenv("TEST_DB_HOST") != "" {
		dbHost = os.Getenv("TEST_DB_HOST")
	}

	dbPort := cfg.Postgres.Port
	if port := os.Getenv("TEST_DB_PORT"); port != "" {
		var err error
		_, err = fmt.Sscanf(port, "%d", &dbPort)
		if err != nil {
			t.Logf("Warning: invalid TEST_DB_PORT, using default: %v", err)
		}
	}

	testDBName := os.Getenv("TEST_DB_NAME")
	if testDBName == "" {
		testDBName = cfg.Postgres.DBName + "_test"
	}

	dbUser := cfg.Postgres.User
	if user := os.Getenv("TEST_DB_USER"); user != "" {
		dbUser = user
	}

	dbPassword := cfg.Postgres.Password
	if password := os.Getenv("TEST_DB_PASSWORD"); password != "" {
		dbPassword = password
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		dbHost, dbUser, dbPassword, testDBName, dbPort,
	)

	t.Logf("Connecting to test DB: host=%s port=%d dbname=%s", dbHost, dbPort, testDBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		db, err = createTestDatabase(testDBName, dbHost, dbUser, dbPassword, dbPort)
		if err != nil {
			t.Fatalf("Failed to create test database: %v", err)
		}
	}

	cleanDB(db)
	if err := db.AutoMigrate(&models.Wallet{}, &models.Transaction{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	t.Cleanup(func() {
		cleanDB(db)
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})

	return db
}

func createTestDatabase(dbName, host, user, password string, port int) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=postgres port=%d sslmode=disable",
		host, user, password, port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	var exists bool
	err = db.Raw("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)", dbName).Scan(&exists).Error
	if err != nil {
		return nil, fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {
		if err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName)).Error; err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
	}

	newDSN := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		host, user, password, dbName, port,
	)

	newDB, err := gorm.Open(postgres.Open(newDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	return newDB, nil
}

func cleanDB(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE transactions RESTART IDENTITY CASCADE")
	db.Exec("TRUNCATE TABLE wallets RESTART IDENTITY CASCADE")
}

func TestGetBalance(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()

	walletID := uuid.New()
	err := repo.UpdateBalance(ctx, walletID, 0, models.Deposit)
	if err != nil {
		t.Fatalf("Failed to create wallet via deposit: %v", err)
	}

	balance, err := repo.GetBalance(ctx, walletID)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	if balance != 0 {
		t.Errorf("Expected balance 0, got %d", balance)
	}
}

func TestGetBalance_WalletNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	fakeID := uuid.New()

	_, err := repo.GetBalance(ctx, fakeID)
	if err == nil {
		t.Error("Expected error for non-existent wallet, got nil")
	}

	if err.Error() != "wallet not found" {
		t.Errorf("Expected 'wallet not found', got '%v'", err)
	}
}

func TestUpdateBalance_Deposit_NewWallet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	walletID := uuid.New()

	err := repo.UpdateBalance(ctx, walletID, 100, models.Deposit)
	if err != nil {
		t.Fatalf("Failed to deposit to new wallet: %v", err)
	}

	balance, err := repo.GetBalance(ctx, walletID)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	if balance != 100 {
		t.Errorf("Expected balance 100, got %d", balance)
	}

	var transaction models.Transaction
	db.Where("wallet_id = ?", walletID).First(&transaction)
	if transaction.Amount != 100 {
		t.Errorf("Expected transaction amount 100, got %d", transaction.Amount)
	}
	if transaction.Type != models.Deposit {
		t.Errorf("Expected transaction type DEPOSIT, got %s", transaction.Type)
	}
}

func TestUpdateBalance_Deposit_ExistingWallet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	walletID := uuid.New()

	err := repo.UpdateBalance(ctx, walletID, 50, models.Deposit)
	if err != nil {
		t.Fatalf("Failed to initial deposit: %v", err)
	}

	err = repo.UpdateBalance(ctx, walletID, 100, models.Deposit)
	if err != nil {
		t.Fatalf("Failed to deposit again: %v", err)
	}

	balance, err := repo.GetBalance(ctx, walletID)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	if balance != 150 {
		t.Errorf("Expected balance 150, got %d", balance)
	}

	var transactions []models.Transaction
	db.Where("wallet_id = ?", walletID).Order("created_at").Find(&transactions)
	if len(transactions) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(transactions))
	}
}

func TestUpdateBalance_Withdraw(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	walletID := uuid.New()

	err := repo.UpdateBalance(ctx, walletID, 200, models.Deposit)
	if err != nil {
		t.Fatalf("Failed to deposit: %v", err)
	}

	err = repo.UpdateBalance(ctx, walletID, 50, models.Withdraw)
	if err != nil {
		t.Fatalf("Failed to withdraw: %v", err)
	}

	balance, err := repo.GetBalance(ctx, walletID)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	if balance != 150 {
		t.Errorf("Expected balance 150, got %d", balance)
	}

	var transaction models.Transaction
	db.Where("wallet_id = ? AND type = ?", walletID, models.Withdraw).First(&transaction)
	if transaction.Amount != 50 {
		t.Errorf("Expected transaction amount 50, got %d", transaction.Amount)
	}
}

func TestUpdateBalance_WithdrawFromNonExistentWallet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	fakeID := uuid.New()

	err := repo.UpdateBalance(ctx, fakeID, 100, models.Withdraw)
	if err == nil {
		t.Error("Expected error for non-existent wallet withdrawal, got nil")
	}

	expectedErr := "wallet not found"
	if err.Error() != expectedErr {
		t.Errorf("Expected '%s', got '%v'", expectedErr, err)
	}
}

func TestUpdateBalance_InsufficientFunds(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	walletID := uuid.New()

	err := repo.UpdateBalance(ctx, walletID, 0, models.Deposit)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	err = repo.UpdateBalance(ctx, walletID, 100, models.Withdraw)
	if err == nil {
		t.Error("Expected insufficient funds error, got nil")
	}

	if err.Error() != "insufficient funds: balance=0, withdraw=100" {
		t.Errorf("Expected insufficient funds error, got '%v'", err)
	}

	balance, _ := repo.GetBalance(ctx, walletID)
	if balance != 0 {
		t.Errorf("Balance should remain 0, got %d", balance)
	}
}

func TestUpdateBalance_ZeroAmount(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	walletID := uuid.New()

	err := repo.UpdateBalance(ctx, walletID, 100, models.Deposit)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	err = repo.UpdateBalance(ctx, walletID, 0, models.Deposit)
	if err != nil {
		t.Fatalf("Failed to deposit zero: %v", err)
	}

	balance, _ := repo.GetBalance(ctx, walletID)
	if balance != 100 {
		t.Errorf("Balance should remain 100, got %d", balance)
	}
}

func TestMultipleOperations(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	walletID := uuid.New()

	operations := []struct {
		opType models.OperationType
		amount int64
	}{
		{models.Deposit, 100},
		{models.Deposit, 50},
		{models.Withdraw, 30},
		{models.Deposit, 200},
		{models.Withdraw, 20},
	}

	expectedBalance := int64(0)
	for _, op := range operations {
		err := repo.UpdateBalance(ctx, walletID, op.amount, op.opType)
		if err != nil {
			t.Fatalf("Operation failed: %+v, error: %v", op, err)
		}
		if op.opType == models.Deposit {
			expectedBalance += op.amount
		} else {
			expectedBalance -= op.amount
		}
	}

	balance, _ := repo.GetBalance(ctx, walletID)
	if balance != expectedBalance {
		t.Errorf("Expected balance %d, got %d", expectedBalance, balance)
	}

	var count int64
	db.Model(&models.Transaction{}).Where("wallet_id = ?", walletID).Count(&count)
	if count != int64(len(operations)) {
		t.Errorf("Expected %d transactions, got %d", len(operations), count)
	}
}

func TestConcurrentDeposits(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	walletID := uuid.New()

	concurrency := 100
	amountPerOp := int64(10)
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			err := repo.UpdateBalance(ctx, walletID, amountPerOp, models.Deposit)
			if err != nil {
				t.Errorf("Concurrent deposit failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}

	balance, _ := repo.GetBalance(ctx, walletID)
	expectedBalance := amountPerOp * int64(concurrency)

	if balance != expectedBalance {
		t.Errorf("Concurrent deposits failed! Expected %d, got %d", expectedBalance, balance)
	}
}

func TestConcurrentMixedOperations(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	walletID := uuid.New()

	err := repo.UpdateBalance(ctx, walletID, 10000, models.Deposit)
	if err != nil {
		t.Fatalf("Failed to initial deposit: %v", err)
	}

	concurrency := 100
	done := make(chan bool, concurrency*2)

	for i := 0; i < concurrency; i++ {
		go func() {
			repo.UpdateBalance(ctx, walletID, 10, models.Deposit)
			done <- true
		}()
		go func() {
			repo.UpdateBalance(ctx, walletID, 5, models.Withdraw)
			done <- true
		}()
	}

	for i := 0; i < concurrency*2; i++ {
		<-done
	}

	balance, _ := repo.GetBalance(ctx, walletID)
	expectedBalance := int64(10000) + (int64(concurrency)*10 - int64(concurrency)*5)

	if balance != expectedBalance {
		t.Errorf("Concurrent mixed operations failed! Expected %d, got %d", expectedBalance, balance)
	}
}

func TestTransactionRollbackOnError(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	walletID := uuid.New()

	err := repo.UpdateBalance(ctx, walletID, 100, models.Deposit)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	err = repo.UpdateBalance(ctx, walletID, 200, models.Withdraw)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	var count int64
	db.Model(&models.Transaction{}).Where("wallet_id = ?", walletID).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 transaction (only deposit), got %d", count)
	}

	balance, _ := repo.GetBalance(ctx, walletID)
	if balance != 100 {
		t.Errorf("Balance should remain 100, got %d", balance)
	}
}

func TestContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	walletID := uuid.New()
	err := repo.UpdateBalance(ctx, walletID, 100, models.Deposit)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}
