package service

import (
	"context"
	"fmt"
	"github.com/Saaghh/wallet/internal/model"
	migrate "github.com/rubenv/sql-migrate"
)

type store interface {
	Migrate(direction migrate.MigrationDirection) error
	GetWalletByID(ctx context.Context, walletID int64) (*model.Wallet, error)
	CreateWallet(ctx context.Context, owner model.User, currency string) (*model.Wallet, error)
	ExecuteTransaction(ctx context.Context, wtx model.Transaction) (*model.Transaction, error)
}

type Service struct {
	db store
}

func New(db store) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) CreateWallet(ctx context.Context, owner model.User, currency string) (*model.Wallet, error) {
	wallet, err := s.db.CreateWallet(ctx, owner, currency)
	if err != nil {
		return nil, fmt.Errorf("s.db.CreateWallet(owner, currency): %w", err)
	}

	return wallet, nil
}

func (s *Service) GetWallet(ctx context.Context, walletID int64) (*model.Wallet, error) {
	wallet, err := s.db.GetWalletByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWalletByID: %w", err)
	}

	return wallet, nil
}

func (s *Service) ExecuteTransaction(ctx context.Context, wtx model.Transaction) (*model.Transaction, error) {
	tx, err := s.db.ExecuteTransaction(ctx, wtx)
	if err != nil {
		return nil, fmt.Errorf("s.db.ExecuteTransaction(ctx, wtx): %w", err)
	}

	return tx, nil
}
