package wallet

import (
	"context"

	"github.com/Black1black/go_base_api/internal/models"
	"github.com/google/uuid"
)

type WalletRepository interface {
	UpdateBalance(ctx context.Context, walletID uuid.UUID, amount int64, operationType models.OperationType) error

	GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error)
}
