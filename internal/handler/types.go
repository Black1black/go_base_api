package handler

import (
	"context"

	"github.com/google/uuid"
)

type (
	WalletUsecase interface {
		ProcessOperation(ctx context.Context, walletID uuid.UUID, operationType string, amount int64) error

		GetWalletBalance(ctx context.Context, walletID uuid.UUID) (int64, error)
	}
)
