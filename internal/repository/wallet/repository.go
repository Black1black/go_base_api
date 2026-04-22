package wallet

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Black1black/go_base_api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db        *gorm.DB
	processor *OperationProcessor
}

func NewRepository(db *gorm.DB) *Repository {
	repo := &Repository{
		db: db,
	}

	repo.processor = NewOperationProcessor(repo)

	return repo
}

func (r *Repository) UpdateBalance(ctx context.Context, walletID uuid.UUID,
	amount int64, operationType models.OperationType) error {

	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	return r.processor.Process(ctx, walletID, amount, operationType)
}

func (r *Repository) updateBalanceTx(ctx context.Context, walletID uuid.UUID,
	amount int64, operationType models.OperationType) error {

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet

		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
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
						if findErr := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
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

func (r *Repository) Shutdown() {
	if r.processor != nil {
		r.processor.Shutdown()
	}
}
