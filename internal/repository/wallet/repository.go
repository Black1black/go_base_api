package wallet

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Black1black/go_base_api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpdateBalance(ctx context.Context, walletID uuid.UUID, amount int64, operationType models.OperationType) error {
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := r.updateBalanceTx(ctx, walletID, amount, operationType)

		if err == nil {
			return nil
		}

		if isRetryableError(err) && attempt < maxRetries-1 {
			backoff := time.Millisecond * 10 * time.Duration(1<<attempt)
			time.Sleep(backoff)
			continue
		}

		return err
	}

	return fmt.Errorf("max retries exceeded for wallet %s", walletID)
}

func (r *Repository) updateBalanceTx(ctx context.Context, walletID uuid.UUID, amount int64, operationType models.OperationType) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet

		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", walletID).
			First(&wallet).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if operationType == models.Deposit {
					wallet = models.Wallet{
						ID:      walletID,
						Balance: 0,
					}
					if createErr := tx.Create(&wallet).Error; createErr != nil {
						if !strings.Contains(createErr.Error(), "duplicate key") {
							return fmt.Errorf("failed to create wallet: %w", createErr)
						}
						if findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
							Where("id = ?", walletID).
							First(&wallet).Error; findErr != nil {
							return fmt.Errorf("failed to find wallet after duplicate: %w", findErr)
						}
					}
				} else {
					return fmt.Errorf("wallet not found")
				}
			} else {
				return fmt.Errorf("failed to lock wallet: %w", err)
			}
		}

		if operationType == models.Withdraw {
			if wallet.Balance < amount {
				return fmt.Errorf("insufficient funds: balance=%d, withdraw=%d", wallet.Balance, amount)
			}
		}

		var newBalance int64
		if operationType == models.Withdraw {
			newBalance = wallet.Balance - amount
		} else {
			newBalance = wallet.Balance + amount
		}

		if err := tx.Model(&models.Wallet{}).
			Where("id = ?", walletID).
			Update("balance", newBalance).Error; err != nil {
			return fmt.Errorf("failed to update balance: %w", err)
		}

		transaction := &models.Transaction{
			WalletID: walletID,
			Type:     operationType,
			Amount:   amount,
		}

		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("failed to create transaction record: %w", err)
		}

		return nil
	})
}

func (r *Repository) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	var wallet models.Wallet

	if err := r.db.WithContext(ctx).
		Where("id = ?", walletID).
		First(&wallet).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("wallet not found")
		}
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	return wallet.Balance, nil
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	deadlockPatterns := []string{
		"deadlock",
		"Deadlock",
		"40P01",
		"balance was modified concurrently",
		"could not serialize",
		"55P03",
	}

	for _, pattern := range deadlockPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}
