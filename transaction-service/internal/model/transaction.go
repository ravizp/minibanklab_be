package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionType string
type TransactionStatus string

const (
	TypeTransfer TransactionType = "TRANSFER"
	TypeTopUp    TransactionType = "TOPUP"

	StatusSuccess TransactionStatus = "SUCCESS"
	StatusFailed  TransactionStatus = "FAILED"
)

type Transaction struct {
	ID              uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	FromAccountID   *uuid.UUID        `gorm:"type:uuid;index" json:"from_account_id"`
	ToAccountNumber string            `gorm:"type:varchar(20);not null" json:"to_account_number"`
	Amount          decimal.Decimal   `gorm:"type:numeric(18,2);not null" json:"amount" swaggertype:"number" example:"100.50"`
	Type            TransactionType   `gorm:"type:varchar(20);not null" json:"type"`
	Status          TransactionStatus `gorm:"type:varchar(20);not null" json:"status"`
	Description     string            `gorm:"type:text" json:"description"`
	CreatedAt       time.Time         `gorm:"autoCreateTime" json:"created_at"`
}

// Account mirrors the accounts table for cross-service balance operations
type Account struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	AccountNumber string          `gorm:"type:varchar(20);uniqueIndex;not null" json:"account_number"`
	UserID        uuid.UUID       `gorm:"type:uuid;index;not null" json:"user_id"`
	Balance       decimal.Decimal `gorm:"type:numeric(18,2);not null;default:0" json:"balance"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Account) TableName() string {
	return "accounts"
}

type TransferRequest struct {
	FromAccountID   string `json:"from_account_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ToAccountNumber string `json:"to_account_number" binding:"required" example:"1234567890"`
	Amount          string `json:"amount" binding:"required" example:"100.50"`
}

type TopUpRequest struct {
	AccountNumber string `json:"account_number" binding:"required" example:"1234567890"`
	Amount        string `json:"amount" binding:"required" example:"500.00"`
}

type TransactionResponse struct {
	ID              uuid.UUID         `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FromAccountID   *uuid.UUID        `json:"from_account_id,omitempty"`
	ToAccountNumber string            `json:"to_account_number" example:"1234567890"`
	Amount          decimal.Decimal   `json:"amount" swaggertype:"number" example:"100.50"`
	Type            TransactionType   `json:"type" example:"TRANSFER"`
	Status          TransactionStatus `json:"status" example:"SUCCESS"`
	Description     string            `json:"description" example:"Transfer 100.50 from 1111111111 to 1234567890"`
	CreatedAt       time.Time         `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"something went wrong"`
}

func (t *Transaction) ToResponse() TransactionResponse {
	return TransactionResponse{
		ID:              t.ID,
		FromAccountID:   t.FromAccountID,
		ToAccountNumber: t.ToAccountNumber,
		Amount:          t.Amount,
		Type:            t.Type,
		Status:          t.Status,
		Description:     t.Description,
		CreatedAt:       t.CreatedAt,
	}
}
