package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"transaction-service/internal/messaging"
	"transaction-service/internal/model"
	"transaction-service/internal/repository"

	"github.com/shopspring/decimal"
)

type TransactionService interface {
	Transfer(ctx context.Context, req model.TransferRequest) (*model.Transaction, error)
	TopUp(ctx context.Context, req model.TopUpRequest) (*model.Transaction, error)
	GetHistory(ctx context.Context, accountID string) ([]model.Transaction, error)
}

type transactionService struct {
	repo repository.TransactionRepository
	mq   *messaging.RabbitMQ
}

func NewTransactionService(repo repository.TransactionRepository, mq *messaging.RabbitMQ) TransactionService {
	return &transactionService{repo: repo, mq: mq}
}

func (s *transactionService) Transfer(ctx context.Context, req model.TransferRequest) (*model.Transaction, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, errors.New("invalid amount format, use numeric string like \"100.50\"")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be greater than zero")
	}

	tx, err := s.repo.Transfer(ctx, req.FromAccountID, req.ToAccountNumber, amount)
	if err != nil {
		return nil, err
	}

	s.publishEvent(ctx, tx)
	return tx, nil
}

func (s *transactionService) TopUp(ctx context.Context, req model.TopUpRequest) (*model.Transaction, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, errors.New("invalid amount format, use numeric string like \"500.00\"")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be greater than zero")
	}

	tx, err := s.repo.TopUp(ctx, req.AccountNumber, amount)
	if err != nil {
		return nil, err
	}

	s.publishEvent(ctx, tx)
	return tx, nil
}

func (s *transactionService) GetHistory(ctx context.Context, accountID string) ([]model.Transaction, error) {
	return s.repo.GetHistoryByAccountID(ctx, accountID)
}

func (s *transactionService) publishEvent(ctx context.Context, tx *model.Transaction) {
	if s.mq == nil {
		return
	}
	data, err := json.Marshal(tx.ToResponse())
	if err != nil {
		log.Printf("[Event] Failed to marshal transaction event: %v", err)
		return
	}
	if err := s.mq.Publish(ctx, "transaction.created", data); err != nil {
		log.Printf("[Event] Failed to publish transaction.created: %v", err)
	} else {
		log.Printf("[Event] Published transaction.created for %s", tx.ID)
	}
}
