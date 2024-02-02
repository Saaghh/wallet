package service

import (
	"context"
	"fmt"
	"github.com/Saaghh/wallet/internal/model"
)

type store interface {
	CreateWallet(ctx context.Context, wallet model.Wallet) (*model.Wallet, error)
	GetWalletByID(ctx context.Context, walletID int64) (*model.Wallet, error)
	GetWallets(ctx context.Context) ([]*model.Wallet, error)
	DeleteWallet(ctx context.Context, walletID int64) error
	UpdateWallet(ctx context.Context, walletID int64, request model.UpdateWalletRequest) (*model.Wallet, error)

	GetTransactions(ctx context.Context) ([]*model.Transaction, error)
	Transfer(ctx context.Context, wtx model.Transaction) (int64, error)
	ExternalTransaction(ctx context.Context, transaction model.Transaction) (int64, error)
}

type Service struct {
	db store
}

func New(db store) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) CreateWallet(ctx context.Context, wallet model.Wallet) (*model.Wallet, error) {
	rWallet, err := s.db.CreateWallet(ctx, wallet)
	if err != nil {
		return nil, fmt.Errorf("s.db.CreateWallet(ctx, owner, currency): %w", err)
	}

	return rWallet, nil
}

func (s *Service) GetWalletByID(ctx context.Context, walletID int64) (*model.Wallet, error) {
	wallet, err := s.db.GetWalletByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWalletByID(ctx, walletID): %w", err)
	}

	return wallet, nil
}

func (s *Service) Transfer(ctx context.Context, transaction model.Transaction) (int64, error) {
	transactionID, err := s.db.Transfer(ctx, transaction)
	if err != nil {
		return 0, fmt.Errorf("s.db.Transfer(ctx, transaction): %w", err)
	}

	return transactionID, nil
}

func (s *Service) ExternalTransaction(ctx context.Context, transaction model.Transaction) (int64, error) {
	transactionID, err := s.db.ExternalTransaction(ctx, transaction)
	if err != nil {
		return 0, fmt.Errorf("s.db.ExternalTransaction(ctx, transaction): %w", err)
	}

	return transactionID, nil
}

func (s *Service) GetWallets(ctx context.Context) ([]*model.Wallet, error) {
	wallets, err := s.db.GetWallets(ctx)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWallets(ctx, owner): %w", err)
	}

	return wallets, err
}

func (s *Service) DeleteWallet(ctx context.Context, walletID int64) error {
	err := s.db.DeleteWallet(ctx, walletID)
	if err != nil {
		return fmt.Errorf("s.db.DeleteWallet(ctx, walletID): %w", err)
	}

	return nil
}

func (s *Service) UpdateWallet(ctx context.Context, walletID int64, request model.UpdateWalletRequest) (*model.Wallet, error) {
	wallet, err := s.db.UpdateWallet(ctx, walletID, request)
	if err != nil {
		return nil, fmt.Errorf("s.db.UpdateWallet(ctx, walletID, request): %w", err)
	}

	return wallet, nil
}

func (s *Service) GetTransactions(ctx context.Context) ([]*model.Transaction, error) {
	transactions, err := s.db.GetTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetTransactions(ctx): %w", err)
	}

	return transactions, nil
}
