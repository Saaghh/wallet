//go:build !MySql
// +build !MySql

package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"time"

	"github.com/Saaghh/wallet/internal/model"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func (p *Postgres) CreateUser(ctx context.Context, user model.User) (*model.User, error) {
	query := `
	INSERT INTO users (email)
	VALUES ($1)
	RETURNING id, registered_at
`

	err := p.db.QueryRow(
		ctx,
		query,
		user.Email,
	).Scan(
		&user.ID,
		&user.RegDate,
	)

	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow(...): %w", err)
	}

	return &user, nil
}

func (p *Postgres) TruncateTables(ctx context.Context) error {

	_, err := p.db.Exec(
		ctx,
		"TRUNCATE TABLE transactions CASCADE")

	if err != nil {
		return fmt.Errorf("p.db.Exec(...): %w", err)
	}

	_, err = p.db.Exec(
		ctx,
		"TRUNCATE TABLE wallets CASCADE")

	if err != nil {
		return fmt.Errorf("p.db.Exec(...): %w", err)
	}

	_, err = p.db.Exec(
		ctx,
		"TRUNCATE TABLE users CASCADE")

	if err != nil {
		return fmt.Errorf("p.db.Exec(...): %w", err)
	}

	return nil
}

func (p *Postgres) CreateWallet(ctx context.Context, wallet model.Wallet) (*model.Wallet, error) {
	if wallet.OwnerID == uuid.Nil {
		return nil, model.ErrNilUUID
	}

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

	// Checking if name is free
	query = `
	SELECT FROM wallets
	WHERE owner_id = $1 and name = $2 and is_disabled = false
`
	err = p.db.QueryRow(
		ctx,
		query,
		wallet.OwnerID, wallet.Name).Scan()

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		break
	case err != nil:
		return nil, fmt.Errorf("p.db.QueryRow(...): %w", err)
	default:
		return nil, model.ErrDuplicateWallet
	}

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

func (p *Postgres) GetWalletByID(ctx context.Context, walletID uuid.UUID) (*model.Wallet, error) {
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

func (p *Postgres) DeleteWallet(ctx context.Context, walletID uuid.UUID) error {

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

func (p *Postgres) UpdateWallet(ctx context.Context, walletID uuid.UUID, request model.UpdateWalletRequest) (*model.Wallet, error) {

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

func (p *Postgres) Transfer(ctx context.Context, transaction model.Transaction) (*uuid.UUID, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("p.db.Begin(ctx): %w", err)
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
		return nil, model.ErrNegativeRequestBalance
	}

	agentWallet, err := p.GetWalletByID(ctx, *transaction.AgentWalletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("p.GetWalletByID(ctx, transaction.AgentWalletID): %w", err)
	}

	if agentWallet.Currency != transaction.Currency {
		return nil, model.ErrWrongCurrency
	}

	if agentWallet.Balance < transaction.Sum {
		return nil, model.ErrNotEnoughBalance
	}

	targetWallet, err := p.GetWalletByID(ctx, *transaction.TargetWalletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("p.GetWalletByID(ctx, transaction.TargetWalletID): %w", err)
	}

	if targetWallet.Currency != transaction.Currency {
		return nil, model.ErrWrongCurrency
	}

	// Saving transaction to DB
	query := `
	INSERT INTO transactions (id, from_wallet_id, to_wallet_id, currency, balance)
	VALUES ($1, $2, $3, $4, $5)
	returning id, created_at
`
	err = tx.QueryRow(
		ctx,
		query,
		transaction.ID, transaction.AgentWalletID, transaction.TargetWalletID, transaction.Currency, transaction.Sum,
	).Scan(
		&transaction.ID,
		&transaction.CreatedAt,
	)

	// Check for unique constraint violation error
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation:
		return nil, model.ErrDuplicateTransaction
	case err != nil:
		return nil, fmt.Errorf("tx.QueryRow(): %w", err)
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
		return nil, fmt.Errorf("tx.Exec(ctx, query, agentWallet.Sum, agentWallet.ID): %w", err)
	}

	targetWallet.Balance += transaction.Sum
	targetWallet.ModifiedDate = time.Now()
	_, err = tx.Exec(
		ctx,
		query,
		targetWallet.Balance, targetWallet.ID, targetWallet.ModifiedDate)

	if err != nil {
		return nil, fmt.Errorf("tx.Exec(ctx, query, targetWallet.Sum, targetWallet.ID): %w", err)
	}

	// Committing transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return &transaction.ID, nil
}

func (p *Postgres) ExternalTransaction(ctx context.Context, transaction model.Transaction) (*uuid.UUID, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("p.db.Begin(ctx): %w", err)
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
		return nil, fmt.Errorf("p.GetWalletByID(ctx, transaction.TargetWalletID): %w", err)
	case errors.Is(err, model.ErrWalletNotFound):
		return nil, err
	}

	// Save transaction
	query := `
	INSERT INTO transactions (id, to_wallet_id, currency, balance)
	VALUES ($1, $2, $3, $4)
	returning id, created_at
`
	err = tx.QueryRow(
		ctx,
		query,
		transaction.ID, transaction.TargetWalletID, transaction.Currency, transaction.Sum,
	).Scan(
		&transaction.ID,
		&transaction.CreatedAt,
	)

	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation:
		return nil, model.ErrDuplicateTransaction
	case err != nil:
		return nil, fmt.Errorf("tx.QueryRow(): %w", err)
	}

	if targetWallet.Currency != transaction.Currency {
		return nil, model.ErrWrongCurrency
	}

	if targetWallet.Balance+transaction.Sum < 0 {
		return nil, model.ErrNotEnoughBalance
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
		return nil, fmt.Errorf("tx.Exec(ctx, query, targetWallet.Sum, targetWallet.ID): %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return &transaction.ID, nil
}

func (p *Postgres) GetTransactionByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {

	if id == uuid.Nil {
		return nil, model.ErrNilUUID
	}

	query := `
	SELECT id, created_at, to_wallet_id, from_wallet_id, currency, balance
	FROM transactions
	WHERE id = $1
`
	var transaction model.Transaction

	err := p.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&transaction.ID,
		&transaction.CreatedAt,
		&transaction.TargetWalletID,
		&transaction.AgentWalletID,
		&transaction.Currency,
		&transaction.Sum,
	)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, model.ErrTransactionsNotFound
	case err != nil:
		return nil, fmt.Errorf("p.db.QueryRow(...): %w", err)
	}

	return &transaction, nil
}
