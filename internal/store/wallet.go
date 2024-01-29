package store

import (
	"context"
	"fmt"
	"github.com/Saaghh/wallet/internal/model"
	"go.uber.org/zap"
	"time"
)

func (p *Postgres) CreateWallet(ctx context.Context, owner model.User, currency string) (*model.Wallet, error) {
	wallet := new(model.Wallet)

	query := `
    INSERT INTO wallets (owner_id, currency)
    VALUES ($1, $2)
    RETURNING id, owner_id, currency, balance, created_at, modified_at
`

	err := p.db.QueryRow(
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

	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow: %w", err)
	}

	return wallet, nil
}

func (p *Postgres) createTransaction(ctx context.Context, wtx model.Transaction) (*model.Transaction, error) {
	query := `
	INSERT INTO transactions (from_wallet_id, to_wallet_id, currency, balance)
	VALUES ($1, $2, $3, $4)
	returning id, created_at
`
	err := p.db.QueryRow(
		ctx,
		query,
		wtx.FromWalletID, wtx.ToWalletID, wtx.Currency, wtx.Balance,
	).Scan(
		&wtx.ID,
		&wtx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow(): %w", err)
	}
	return &wtx, nil
}

func (p *Postgres) ExecuteTransaction(ctx context.Context, wtx model.Transaction) (*model.Transaction, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("p.db.Begin(ctx): %w", err)
	}
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil {
			zap.L().With(zap.Error(err)).Warn("ExecuteTransaction/tx.Rollback(ctx)")
		}
	}()

	//Saving transaction to DB
	savedTX, err := p.createTransaction(ctx, wtx)

	//Checking Balance and currency
	fromWallet, err := p.GetWalletByID(ctx, savedTX.FromWalletID)
	if err != nil {
		return nil, fmt.Errorf("p.GetWalletByID(ctx, savedTX.FromWalletID): %w", err)
	}
	if fromWallet.Currency != savedTX.Currency {
		return nil, fmt.Errorf("wrong currency fromWallet")
	}
	if fromWallet.Balance < savedTX.Balance {
		return nil, fmt.Errorf("not enough balance fromWallet")
	}

	toWallet, err := p.GetWalletByID(ctx, savedTX.ToWalletID)
	if err != nil {
		return nil, fmt.Errorf("p.GetWalletByID(ctx, savedTX.ToWalletID): %w", err)
	}
	if toWallet.Currency != savedTX.Currency {
		return nil, fmt.Errorf("wrong currency toWallet")
	}

	//Moving Cash
	query := `
	UPDATE wallets
	SET balance = $1, modified_at = $3
	WHERE id = $2
`

	fromWallet.Balance -= savedTX.Balance
	fromWallet.ModifiedDate = time.Now()
	_, err = tx.Exec(
		ctx,
		query,
		fromWallet.Balance, fromWallet.ID, fromWallet.ModifiedDate)
	if err != nil {
		return nil, fmt.Errorf("tx.Exec(ctx, query, fromWallet.Balance, fromWallet.ID): %w", err)
	}

	toWallet.Balance += savedTX.Balance
	toWallet.ModifiedDate = time.Now()
	_, err = tx.Exec(
		ctx,
		query,
		toWallet.Balance, toWallet.ID, toWallet.ModifiedDate)
	if err != nil {
		return nil, fmt.Errorf("tx.Exec(ctx, query, toWallet.Balance, toWallet.ID): %w", err)
	}

	//Confirming transaction

	query = `
	UPDATE transactions
	SET finished_at = $1
	WHERE id = $2
`
	savedTX.FinishedAt = time.Now()
	_, err = tx.Exec(
		ctx,
		query,
		savedTX.FinishedAt, savedTX.ID)

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return savedTX, nil

}
