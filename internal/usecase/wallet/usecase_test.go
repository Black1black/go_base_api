package wallet

import (
	"context"
	"errors"
	"testing"

	"github.com/Black1black/go_base_api/internal/models"
	"github.com/google/uuid"
)

type mockRepository struct {
	updateBalanceFunc func(ctx context.Context, walletID uuid.UUID, amount int64, operationType models.OperationType) error
	getBalanceFunc    func(ctx context.Context, walletID uuid.UUID) (int64, error)
	createWalletFunc  func(ctx context.Context) (*models.Wallet, error)
}

func (m *mockRepository) UpdateBalance(ctx context.Context, walletID uuid.UUID, amount int64, operationType models.OperationType) error {
	return m.updateBalanceFunc(ctx, walletID, amount, operationType)
}

func (m *mockRepository) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	return m.getBalanceFunc(ctx, walletID)
}

func (m *mockRepository) CreateWallet(ctx context.Context) (*models.Wallet, error) {
	return m.createWalletFunc(ctx)
}

func TestUsecase_ProcessOperation_InvalidAmount(t *testing.T) {
	repo := &mockRepository{}
	uc := NewUsecase(repo)

	walletID := uuid.New()
	err := uc.ProcessOperation(context.Background(), walletID, "DEPOSIT", -100)

	if err == nil {
		t.Error("Expected error for negative amount, got nil")
	}
	if err.Error() != "amount must be positive" {
		t.Errorf("Expected 'amount must be positive', got %v", err)
	}
}

func TestUsecase_ProcessOperation_InvalidOperationType(t *testing.T) {
	repo := &mockRepository{}
	uc := NewUsecase(repo)

	walletID := uuid.New()
	err := uc.ProcessOperation(context.Background(), walletID, "INVALID", 100)

	if err == nil {
		t.Error("Expected error for invalid operation type, got nil")
	}
}

func TestUsecase_ProcessOperation_DepositSuccess(t *testing.T) {
	repo := &mockRepository{
		updateBalanceFunc: func(ctx context.Context, walletID uuid.UUID, amount int64, operationType models.OperationType) error {
			return nil
		},
	}
	uc := NewUsecase(repo)

	walletID := uuid.New()
	err := uc.ProcessOperation(context.Background(), walletID, "DEPOSIT", 100)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestUsecase_ProcessOperation_WithdrawSuccess(t *testing.T) {
	repo := &mockRepository{
		updateBalanceFunc: func(ctx context.Context, walletID uuid.UUID, amount int64, operationType models.OperationType) error {
			return nil
		},
	}
	uc := NewUsecase(repo)

	walletID := uuid.New()
	err := uc.ProcessOperation(context.Background(), walletID, "WITHDRAW", 100)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestUsecase_ProcessOperation_WalletNotFound(t *testing.T) {
	repo := &mockRepository{
		updateBalanceFunc: func(ctx context.Context, walletID uuid.UUID, amount int64, operationType models.OperationType) error {
			return errors.New("wallet not found")
		},
	}
	uc := NewUsecase(repo)

	walletID := uuid.New()
	err := uc.ProcessOperation(context.Background(), walletID, "DEPOSIT", 100)

	if err == nil || err.Error() != "wallet not found" {
		t.Errorf("Expected 'wallet not found', got %v", err)
	}
}

func TestUsecase_GetWalletBalance_Success(t *testing.T) {
	expectedBalance := int64(500)
	repo := &mockRepository{
		getBalanceFunc: func(ctx context.Context, walletID uuid.UUID) (int64, error) {
			return expectedBalance, nil
		},
	}
	uc := NewUsecase(repo)

	walletID := uuid.New()
	balance, err := uc.GetWalletBalance(context.Background(), walletID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if balance != expectedBalance {
		t.Errorf("Expected balance %d, got %d", expectedBalance, balance)
	}
}

func TestUsecase_GetWalletBalance_NotFound(t *testing.T) {
	repo := &mockRepository{
		getBalanceFunc: func(ctx context.Context, walletID uuid.UUID) (int64, error) {
			return 0, errors.New("wallet not found")
		},
	}
	uc := NewUsecase(repo)

	walletID := uuid.New()
	_, err := uc.GetWalletBalance(context.Background(), walletID)

	if err == nil || err.Error() != "wallet not found" {
		t.Errorf("Expected 'wallet not found', got %v", err)
	}
}
