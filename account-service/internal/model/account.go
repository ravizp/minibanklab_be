package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Account struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AccountNumber string          `gorm:"type:varchar(20);uniqueIndex;not null" json:"account_number"`
	UserID        uuid.UUID       `gorm:"type:uuid;index;not null" json:"user_id"`
	Balance       decimal.Decimal `gorm:"type:numeric(18,2);not null;default:0" json:"balance" swaggertype:"number" example:"1000.50"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

type AccountResponse struct {
	ID            uuid.UUID       `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountNumber string          `json:"account_number" example:"1234567890"`
	UserID        uuid.UUID       `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Balance       decimal.Decimal `json:"balance" swaggertype:"number" example:"1000.50"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// BalanceResponse returns only the balance-related fields for an account.
type BalanceResponse struct {
	AccountID     uuid.UUID       `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountNumber string          `json:"account_number" example:"1234567890"`
	Balance       decimal.Decimal `json:"balance" swaggertype:"number" example:"1000.50"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"something went wrong"`
}

func (a *Account) ToResponse() AccountResponse {
	return AccountResponse{
		ID:            a.ID,
		AccountNumber: a.AccountNumber,
		UserID:        a.UserID,
		Balance:       a.Balance,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

// ToBalanceResponse converts an Account to a BalanceResponse.
func (a *Account) ToBalanceResponse() BalanceResponse {
	return BalanceResponse{
		AccountID:     a.ID,
		AccountNumber: a.AccountNumber,
		Balance:       a.Balance,
	}
}
