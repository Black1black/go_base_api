package wallet

import (
	"context"
	"errors"
	"fmt"

	"github.com/Black1black/go_base_api/internal/models"
	"github.com/google/uuid"
	"github.com/ybru-tech/georm"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpdateBalance(ctx context.Context, walletID uuid.UUID, amount int64, operationType models.OperationType) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet

		if err := tx.Clauses(georm.Locking{Strength: "UPDATE"}).
			Where("id = ?", walletID).
			First(&wallet).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("wallet not found")
			}
			return err
		}

		if operationType == models.Withdraw {
			if wallet.Balance < amount {
				return fmt.Errorf("insufficient funds: balance=%d, withdraw=%d", wallet.Balance, amount)
			}
			wallet.Balance -= amount
		} else {
			wallet.Balance += amount
		}

		if err := tx.Save(&wallet).Error; err != nil {
			return err
		}

		transaction := &models.Transaction{
			WalletID: walletID,
			Type:     operationType,
			Amount:   amount,
		}

		if err := tx.Create(transaction).Error; err != nil {
			return err
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
		return 0, err
	}

	return wallet.Balance, nil
}

func (r *Repository) CreateWallet(ctx context.Context) (*models.Wallet, error) {
	wallet := &models.Wallet{
		Balance: 0,
	}

	if err := r.db.WithContext(ctx).Create(wallet).Error; err != nil {
		return nil, err
	}

	return wallet, nil
}
