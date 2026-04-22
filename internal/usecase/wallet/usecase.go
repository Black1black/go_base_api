package wallet

import (
	"context"
	"fmt"

	"github.com/Black1black/go_base_api/internal/models"
	"github.com/google/uuid"
)

type Usecase struct {
	repo WalletRepository
}

func NewUsecase(repo WalletRepository) *Usecase {
	return &Usecase{
		repo: repo,
	}
}

func (s *Usecase) ProcessOperation(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	var opType models.OperationType
	switch operationType {
	case "DEPOSIT":
		opType = models.Deposit
	case "WITHDRAW":
		opType = models.Withdraw
	default:
		return fmt.Errorf("invalid operation type")
	}

	return s.repo.UpdateBalance(ctx, walletID, amount, opType)
}

func (s *Usecase) GetWalletBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {

	return s.repo.GetBalance(ctx, walletID)
}
