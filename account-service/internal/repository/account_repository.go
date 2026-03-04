package repository

import (
	"context"

	"account-service/internal/model"

	"gorm.io/gorm"
)

type AccountRepository interface {
	Create(ctx context.Context, account *model.Account) error
	FindByID(ctx context.Context, id string) (*model.Account, error)
	FindByUserID(ctx context.Context, userID string) ([]model.Account, error)
	FindByAccountNumber(ctx context.Context, number string) (*model.Account, error)
}

type accountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Create(ctx context.Context, account *model.Account) error {
	return r.db.WithContext(ctx).Create(account).Error
}

func (r *accountRepository) FindByID(ctx context.Context, id string) (*model.Account, error) {
	var account model.Account
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *accountRepository) FindByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	var accounts []model.Account
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *accountRepository) FindByAccountNumber(ctx context.Context, number string) (*model.Account, error) {
	var account model.Account
	if err := r.db.WithContext(ctx).Where("account_number = ?", number).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}
