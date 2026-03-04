package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"account-service/internal/model"
	"account-service/internal/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountService interface {
	CreateAccount(ctx context.Context, userID string) (*model.Account, error)
	GetAccountByID(ctx context.Context, id string) (*model.Account, error)
	GetAccountsByUserID(ctx context.Context, userID string) ([]model.Account, error)
}

type accountService struct {
	repo repository.AccountRepository
}

func NewAccountService(repo repository.AccountRepository) AccountService {
	return &accountService{repo: repo}
}

func generateAccountNumber() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(9000000000))
	return fmt.Sprintf("%010d", n.Int64()+1000000000)
}

func (s *accountService) CreateAccount(ctx context.Context, userID string) (*model.Account, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	account := &model.Account{
		ID:            uuid.New(),
		AccountNumber: generateAccountNumber(),
		UserID:        uid,
		Balance:       decimal.Zero,
	}

	if err := s.repo.Create(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (s *accountService) GetAccountByID(ctx context.Context, id string) (*model.Account, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *accountService) GetAccountsByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	return s.repo.FindByUserID(ctx, userID)
}
