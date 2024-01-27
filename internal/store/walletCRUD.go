package store

import (
	"context"
	"fmt"
	"github.com/Saaghh/wallet/internal/model"
	"time"
)

func (p *Postgres) CreateWallet(ctx context.Context, owner model.User, currency string) (int64, error) {
	var walletID int64
	currentDate := time.Now()

	err := p.db.QueryRow(ctx, "INSERT INTO wallets (ownerid, currency, balance, createddate, modifieddate) VALUES ($1, $2, $3, $4, $5) RETURNING id", owner.ID, currency, 0, currentDate, currentDate).Scan(&walletID)
	if err != nil {
		return 0, fmt.Errorf("p.db.QueryRow(): %w", err)
	}

	return walletID, nil
}
