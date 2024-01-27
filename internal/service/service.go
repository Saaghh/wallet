package service

import (
	"context"
	"fmt"
	"github.com/Saaghh/wallet/internal/model"
	"github.com/Saaghh/wallet/internal/store"
)

type Service struct {
	visitHistory map[string]int
	db           *store.Postgres
}

func New(db *store.Postgres) *Service {
	history := make(map[string]int)

	return &Service{
		visitHistory: history,
		db:           db,
	}
}

func (s *Service) SaveVisit(addr string) {
	s.visitHistory[addr]++
}

func (s *Service) GetVisitHistory() map[string]int {
	return s.visitHistory
}

func (s *Service) CreateUser(ctx context.Context, email string) (*model.User, error) {
	user, err := s.db.CreateUser(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("s.db.CreateUser(email): %w", err)
	}

	return user, nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetUserByEmail(email): %w", err)
	}

	return user, nil
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
