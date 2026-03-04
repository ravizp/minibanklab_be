package repository

import (
	"context"
	"errors"
	"fmt"

	"transaction-service/internal/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransactionRepository interface {
	Transfer(ctx context.Context, fromAccountID, toAccountNumber string, amount decimal.Decimal) (*model.Transaction, error)
	TopUp(ctx context.Context, accountNumber string, amount decimal.Decimal) (*model.Transaction, error)
	GetHistoryByAccountID(ctx context.Context, accountID string) ([]model.Transaction, error)
}

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Transfer(ctx context.Context, fromAccountID, toAccountNumber string, amount decimal.Decimal) (*model.Transaction, error) {
	var txRecord model.Transaction

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var fromAccount model.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", fromAccountID).First(&fromAccount).Error; err != nil {
			return errors.New("source account not found")
		}

		if fromAccount.Balance.LessThan(amount) {
			return errors.New("insufficient balance")
		}

		var toAccount model.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_number = ?", toAccountNumber).First(&toAccount).Error; err != nil {
			return errors.New("destination account not found")
		}

		if fromAccount.ID == toAccount.ID {
			return errors.New("cannot transfer to the same account")
		}

		newFromBalance := fromAccount.Balance.Sub(amount)
		if newFromBalance.LessThan(decimal.Zero) {
			return errors.New("transfer would result in negative balance")
		}

		if err := tx.Model(&fromAccount).Update("balance", newFromBalance).Error; err != nil {
			return fmt.Errorf("failed to debit source: %w", err)
		}
		newToBalance := toAccount.Balance.Add(amount)
		if err := tx.Model(&toAccount).Update("balance", newToBalance).Error; err != nil {
			return fmt.Errorf("failed to credit destination: %w", err)
		}

		fromID := uuid.MustParse(fromAccountID)
		txRecord = model.Transaction{
			ID:              uuid.New(),
			FromAccountID:   &fromID,
			ToAccountNumber: toAccountNumber,
			Amount:          amount,
			Type:            model.TypeTransfer,
			Status:          model.StatusSuccess,
			Description:     fmt.Sprintf("Transfer %s from %s to %s", amount.StringFixed(2), fromAccount.AccountNumber, toAccountNumber),
		}

		return tx.Create(&txRecord).Error
	})

	if err != nil {
		return nil, err
	}
	return &txRecord, nil
}

func (r *transactionRepository) TopUp(ctx context.Context, accountNumber string, amount decimal.Decimal) (*model.Transaction, error) {
	var txRecord model.Transaction

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account model.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_number = ?", accountNumber).First(&account).Error; err != nil {
			return errors.New("account not found")
		}

		newBalance := account.Balance.Add(amount)
		if err := tx.Model(&account).Update("balance", newBalance).Error; err != nil {
			return fmt.Errorf("failed to credit account: %w", err)
		}

		txRecord = model.Transaction{
			ID:              uuid.New(),
			ToAccountNumber: accountNumber,
			Amount:          amount,
			Type:            model.TypeTopUp,
			Status:          model.StatusSuccess,
			Description:     fmt.Sprintf("Top up %s to account %s", amount.StringFixed(2), accountNumber),
		}

		return tx.Create(&txRecord).Error
	})

	if err != nil {
		return nil, err
	}
	return &txRecord, nil
}

func (r *transactionRepository) GetHistoryByAccountID(ctx context.Context, accountID string) ([]model.Transaction, error) {
	var account model.Account
	if err := r.db.WithContext(ctx).Where("id = ?", accountID).First(&account).Error; err != nil {
		return nil, errors.New("account not found")
	}

	var transactions []model.Transaction
	err := r.db.WithContext(ctx).
		Where("from_account_id = ? OR to_account_number = ?", accountID, account.AccountNumber).
		Order("created_at DESC").
		Find(&transactions).Error

	return transactions, err
}
