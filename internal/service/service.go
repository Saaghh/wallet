package service

import (
	"context"

	"github.com/Saaghh/wallet/internal/model"
	migrate "github.com/rubenv/sql-migrate"
)

type store interface {
	Migrate(direction migrate.MigrationDirection) error
	GetWalletByID(ctx context.Context, walletID int64) (*model.Wallet, error)
	CreateWallet(ctx context.Context, owner model.User, currency string) (*model.Wallet, error)
	Transfer(ctx context.Context, wtx model.Transaction) (int64, error)
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
		return nil, err
	}

	return wallet, nil
}

func (s *Service) GetWallet(ctx context.Context, walletID int64) (*model.Wallet, error) {
	wallet, err := s.db.GetWalletByID(ctx, walletID)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

func (s *Service) Transfer(ctx context.Context, wtx model.Transaction) (int64, error) {
	transferID, err := s.db.Transfer(ctx, wtx)
	if err != nil {
		return 0, err
	}

	return transferID, nil
}
