package models

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Balance   int64     `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type OperationType string

const (
	Deposit  OperationType = "DEPOSIT"
	Withdraw OperationType = "WITHDRAW"
)

type Transaction struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WalletID  uuid.UUID     `gorm:"type:uuid;not null;index"`
	Type      OperationType `gorm:"type:varchar(10);not null"`
	Amount    int64         `gorm:"not null"`
	CreatedAt time.Time
}
