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

func (p *Postgres) CreateWallet(ctx context.Context, wallet model.Wallet) (*model.Wallet, error) {
	// Checking if user exists
	query := `
	SELECT FROM users
	WHERE id = $1
`
	err := p.db.QueryRow(
		ctx,
		query,
		wallet.OwnerID,
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
    INSERT INTO wallets (owner_id, currency, name)
    VALUES ($1, $2, $3)
    RETURNING id, owner_id, currency, balance, created_at, modified_at
`

	err = p.db.QueryRow(
		ctx,
		query,
		wallet.OwnerID, wallet.Currency, wallet.Name,
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

	return &wallet, nil
}

func (p *Postgres) GetWalletByID(ctx context.Context, walletID int64) (*model.Wallet, error) {
	wallet := new(model.Wallet)
	query := `
	SELECT id, owner_id, currency, balance, created_at, modified_at, name
	FROM wallets
	WHERE id = $1 AND is_disabled = false
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
		&wallet.Name,
	)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, model.ErrWalletNotFound
	case err != nil:
		return nil, fmt.Errorf("p.db.QueryRow: %w", err)
	}

	return wallet, nil
}

func (p *Postgres) GetTransactions(ctx context.Context) ([]*model.Transaction, error) {
	transactions := make([]*model.Transaction, 0, 1)

	query := `
	SELECT id, from_wallet_id, to_wallet_id, currency, balance, created_at
	FROM transactions
`
	rows, err := p.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("p.db.Query(ctx, query): %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		transaction := new(model.Transaction)

		err := rows.Scan(
			&transaction.ID,
			&transaction.AgentWalletID,
			&transaction.TargetWalletID,
			&transaction.Currency,
			&transaction.Sum,
			&transaction.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("rows.Scan(...): %w", err)
		}

		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %w", err)
	}

	if len(transactions) == 0 {
		return nil, model.ErrTransactionsNotFound
	}

	return transactions, nil

}

func (p *Postgres) GetWallets(ctx context.Context) ([]*model.Wallet, error) {
	wallets := make([]*model.Wallet, 0, 1)

	query := `
	SELECT id, owner_id, currency, balance, created_at, modified_at, name
	FROM wallets
	WHERE is_disabled = false
`
	rows, err := p.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("p.db.Query(ctx, query, owner.ID): %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		wallet := new(model.Wallet)

		err = rows.Scan(
			&wallet.ID,
			&wallet.OwnerID,
			&wallet.Currency,
			&wallet.Balance,
			&wallet.CreatedDate,
			&wallet.ModifiedDate,
			&wallet.Name)
		if err != nil {
			return nil, fmt.Errorf("err = rows.Scan(...): %w", err)
		}

		wallets = append(wallets, wallet)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %w", err)
	}

	if len(wallets) == 0 {
		return nil, model.ErrWalletNotFound
	}

	return wallets, nil
}

func (p *Postgres) DeleteWallet(ctx context.Context, walletID int64) error {

	_, err := p.GetWalletByID(ctx, walletID)
	switch {
	case errors.Is(err, model.ErrWalletNotFound):
		return model.ErrWalletNotFound
	case err != nil:
		return fmt.Errorf("p.GetWalletByID(ctx, walletID): %w", err)
	}

	query := `
	UPDATE wallets
	SET is_disabled = true, modified_at = $2
	WHERE id = $1 AND is_disabled = false
`
	err = p.db.QueryRow(
		ctx,
		query,
		walletID, time.Now(),
	).Scan()

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("p.db.QueryRow(...): %w", err)
	}

	return nil
}

func (p *Postgres) UpdateWallet(ctx context.Context, walletID int64, request model.UpdateWalletRequest) (*model.Wallet, error) {

	tx, err := p.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("p.db.Begin(ctx): %w", err)
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			zap.L().With(zap.Error(err)).Warn("UpdateWallet/tx.Rollback(ctx)")
		}
	}()

	if request.Name != nil {
		query := `
		UPDATE wallets
		SET name = $2, modified_at = $3
		WHERE id = $1 AND is_disabled = false
		RETURNING id
	`
		err = tx.QueryRow(
			ctx,
			query,
			walletID, request.Name, time.Now(),
		).Scan(nil)

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, model.ErrWalletNotFound
		case err != nil:
			return nil, fmt.Errorf("p.db.QueryRow(...): %w", err)
		}
	}
	if request.Currency != nil {
		query := `
		UPDATE wallets
		SET currency = $2, modified_at = $3
		WHERE id = $1 AND is_disabled = false
		RETURNING id
	`
		err = tx.QueryRow(
			ctx,
			query,
			walletID, request.Currency, time.Now(),
		).Scan(nil)

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, model.ErrWalletNotFound
		case err != nil:
			return nil, fmt.Errorf("p.db.QueryRow(...): %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	wallet, err := p.GetWalletByID(ctx, walletID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, model.ErrWalletNotFound
	case err != nil:
		return nil, fmt.Errorf("p.db.QueryRow(...): %w", err)
	}

	return wallet, nil
}

func (p *Postgres) Transfer(ctx context.Context, transaction model.Transaction) (int64, error) {
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
	if transaction.Sum < 0 {
		return 0, model.ErrNegativeRequestBalance
	}

	agentWallet, err := p.GetWalletByID(ctx, *transaction.AgentWalletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, model.ErrWalletNotFound
	}

	if err != nil {
		return 0, fmt.Errorf("p.GetWalletByID(ctx, transaction.AgentWalletID): %w", err)
	}

	if agentWallet.Currency != transaction.Currency {
		return 0, model.ErrWrongCurrency
	}

	if agentWallet.Balance < transaction.Sum {
		return 0, model.ErrNotEnoughBalance
	}

	targetWallet, err := p.GetWalletByID(ctx, *transaction.TargetWalletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, model.ErrWalletNotFound
	}

	if err != nil {
		return 0, fmt.Errorf("p.GetWalletByID(ctx, transaction.TargetWalletID): %w", err)
	}

	if targetWallet.Currency != transaction.Currency {
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
		transaction.AgentWalletID, transaction.TargetWalletID, transaction.Currency, transaction.Sum,
	).Scan(
		&transaction.ID,
		&transaction.CreatedAt,
	)

	if err != nil {
		return 0, fmt.Errorf("tx.QueryRow(): %w", err)
	}

	// Moving Cash
	query = `
	UPDATE wallets
	SET balance = $1, modified_at = $3
	WHERE id = $2 
`

	agentWallet.Balance -= transaction.Sum
	agentWallet.ModifiedDate = time.Now()
	_, err = tx.Exec(
		ctx,
		query,
		agentWallet.Balance, agentWallet.ID, agentWallet.ModifiedDate)

	if err != nil {
		return 0, fmt.Errorf("tx.Exec(ctx, query, agentWallet.Sum, agentWallet.ID): %w", err)
	}

	targetWallet.Balance += transaction.Sum
	targetWallet.ModifiedDate = time.Now()
	_, err = tx.Exec(
		ctx,
		query,
		targetWallet.Balance, targetWallet.ID, targetWallet.ModifiedDate)

	if err != nil {
		return 0, fmt.Errorf("tx.Exec(ctx, query, targetWallet.Sum, targetWallet.ID): %w", err)
	}

	// Committing transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return transaction.ID, nil
}

func (p *Postgres) ExternalTransaction(ctx context.Context, transaction model.Transaction) (int64, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("p.db.Begin(ctx): %w", err)
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			zap.L().With(zap.Error(err)).Warn("ExternalTransaction/tx.Rollback(ctx)")
		}
	}()

	// Validate data
	// correct currency
	// wallet exists
	targetWallet, err := p.GetWalletByID(ctx, *transaction.TargetWalletID)

	switch {
	case err != nil:
		return 0, fmt.Errorf("p.GetWalletByID(ctx, transaction.TargetWalletID): %w", err)
	case errors.Is(err, model.ErrWalletNotFound):
		return 0, err
	}

	if targetWallet.Currency != transaction.Currency {
		return 0, model.ErrWrongCurrency
	}

	if targetWallet.Balance+transaction.Sum < 0 {
		return 0, model.ErrNotEnoughBalance
	}

	// Save transaction
	query := `
	INSERT INTO transactions (to_wallet_id, currency, balance)
	VALUES ($1, $2, $3)
	returning id, created_at
`
	err = tx.QueryRow(
		ctx,
		query,
		transaction.TargetWalletID, transaction.Currency, transaction.Sum,
	).Scan(
		&transaction.ID,
		&transaction.CreatedAt,
	)

	if err != nil {
		return 0, fmt.Errorf("tx.QueryRow(): %w", err)
	}
	// Update balance

	query = `
	UPDATE wallets
	SET balance = $1, modified_at = $3
	WHERE id = $2
`

	targetWallet.Balance += transaction.Sum
	targetWallet.ModifiedDate = time.Now()
	_, err = tx.Exec(
		ctx,
		query,
		targetWallet.Balance, targetWallet.ID, targetWallet.ModifiedDate)

	if err != nil {
		return 0, fmt.Errorf("tx.Exec(ctx, query, targetWallet.Sum, targetWallet.ID): %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return transaction.ID, nil
}
