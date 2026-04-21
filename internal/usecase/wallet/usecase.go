package wallet

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Black1black/go_base_api/internal/models"
	"github.com/google/uuid"
)

type Usecase struct {
	repo WalletRepository
	mu   sync.RWMutex
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

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		err := s.repo.UpdateBalance(ctx, walletID, amount, opType)
		if err == nil {
			return nil
		}

		if !isDeadlockError(err) && i == maxRetries-1 {
			return err
		}

		time.Sleep(time.Millisecond * 100 * time.Duration(1<<i))
	}

	return fmt.Errorf("failed after %d retries", maxRetries)
}

func (s *Usecase) GetWalletBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return s.repo.GetBalance(ctx, walletID)
}

func isDeadlockError(err error) bool {
	return err != nil && (err.Error() == "deadlock detected" ||
		err.Error() == "pq: deadlock detected")
}
