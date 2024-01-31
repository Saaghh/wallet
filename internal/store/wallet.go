package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Saaghh/wallet/internal/model"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func (p *Postgres) CreateWallet(ctx context.Context, owner model.User, currency string) (*model.Wallet, error) {
	wallet := new(model.Wallet)

	// Checking if user exists
	query := `
	SELECT FROM users
	WHERE id = $1
`
	err := p.db.QueryRow(
		ctx,
		query,
		owner.ID,
	).Scan()

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, model.ErrUserNotFound
	case err != nil:
		return nil, fmt.Errorf("p.db.QueryRow(): %w", err)
	}
	// TODO Checking if currency is valid

	// Creating wallet
	query = `
    INSERT INTO wallets (owner_id, currency)
    VALUES ($1, $2)
    RETURNING id, owner_id, currency, balance, created_at, modified_at
`

	err = p.db.QueryRow(
		ctx,
		query,
		owner.ID, currency,
	).Scan(
		&wallet.ID,
		&wallet.OwnerID,
		&wallet.Currency,
		&wallet.Balance,
		&wallet.CreatedDate,
		&wallet.ModifiedDate,
	)

	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow(): %w", err)
	}

	return wallet, nil
}

func (p *Postgres) GetWalletByID(ctx context.Context, walletID int64) (*model.Wallet, error) {
	wallet := new(model.Wallet)
	query := `
	SELECT id, owner_id, currency, balance, created_at, modified_at 
	FROM wallets
	WHERE id = $1
`
	err := p.db.QueryRow(
		ctx,
		query,
		walletID,
	).Scan(
		&wallet.ID,
		&wallet.OwnerID,
		&wallet.Currency,
		&wallet.Balance,
		&wallet.CreatedDate,
		&wallet.ModifiedDate,
	)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, model.ErrWalletNotFound
	case err != nil:
		return nil, fmt.Errorf("p.db.QueryRow: %w", err)
	}

	return wallet, nil
}

func (p *Postgres) Transfer(ctx context.Context, transfer model.Transaction) (int64, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("p.db.Begin(ctx): %w", err)
	}
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			zap.L().With(zap.Error(err)).Warn("Transfer/tx.Rollback(ctx)")
		}
	}()

	// TODO Validating all data

	// Verifying data
	// TODO change to switch structure
	if transfer.Balance < 0 {
		return 0, model.ErrNegativeRequestBalance
	}

	agentWallet, err := p.GetWalletByID(ctx, transfer.AgentWalletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, model.ErrWalletNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("p.GetWalletByID(ctx, transfer.AgentWalletID): %w", err)
	}
	if agentWallet.Currency != transfer.Currency {
		return 0, model.ErrWrongCurrency
	}
	if agentWallet.Balance < transfer.Balance {
		return 0, model.ErrNotEnoughBalance
	}

	targetWallet, err := p.GetWalletByID(ctx, transfer.TargetWalletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, model.ErrWalletNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("p.GetWalletByID(ctx, transfer.TargetWalletID): %w", err)
	}
	if targetWallet.Currency != transfer.Currency {
		return 0, model.ErrWrongCurrency
	}

	// Saving transaction to DB
	query := `
	INSERT INTO transactions (from_wallet_id, to_wallet_id, currency, balance)
	VALUES ($1, $2, $3, $4)
	returning id, created_at
`
	err = tx.QueryRow(
		ctx,
		query,
		transfer.AgentWalletID, transfer.TargetWalletID, transfer.Currency, transfer.Balance,
	).Scan(
		&transfer.ID,
		&transfer.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("p.db.QueryRow(): %w", err)
	}

	// Moving Cash
	query = `
	UPDATE wallets
	SET balance = $1, modified_at = $3
	WHERE id = $2
`

	agentWallet.Balance -= transfer.Balance
	agentWallet.ModifiedDate = time.Now()
	_, err = tx.Exec(
		ctx,
		query,
		agentWallet.Balance, agentWallet.ID, agentWallet.ModifiedDate)
	if err != nil {
		return 0, fmt.Errorf("tx.Exec(ctx, query, agentWallet.Balance, agentWallet.ID): %w", err)
	}

	targetWallet.Balance += transfer.Balance
	targetWallet.ModifiedDate = time.Now()
	_, err = tx.Exec(
		ctx,
		query,
		targetWallet.Balance, targetWallet.ID, targetWallet.ModifiedDate)
	if err != nil {
		return 0, fmt.Errorf("tx.Exec(ctx, query, targetWallet.Balance, targetWallet.ID): %w", err)
	}

	// Committing transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return transfer.ID, nil
}
