package store

import (
	"context"
	"fmt"
	"github.com/Saaghh/wallet/internal/model"
)

func (p *Postgres) CreateWallet(ctx context.Context, owner model.User, currency string) (*model.Wallet, error) {
	wallet := new(model.Wallet)

	err := p.db.QueryRow(ctx, "INSERT INTO wallets (owner_id, currency) VALUES ($1, $2) RETURNING id, owner_id, currency, balance, created_at, modified_at", owner.ID, currency).Scan(&wallet.ID, &wallet.OwnerID, &wallet.Currency, &wallet.Balance, &wallet.CreatedDate, &wallet.ModifiedDate)
	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow(): %w", err)
	}

	return wallet, nil
}

func (p *Postgres) GetWalletByID(ctx context.Context, walletID int64) (*model.Wallet, error) {
	wallet := new(model.Wallet)

	err := p.db.QueryRow(ctx, "SELECT id, owner_id, currency, balance, created_at, modified_at FROM wallets WHERE id = $1", walletID).Scan(&wallet.ID, &wallet.OwnerID, &wallet.Currency, &wallet.Balance, &wallet.CreatedDate, &wallet.ModifiedDate)
	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow: %w", err)
	}

	return wallet, nil
}
