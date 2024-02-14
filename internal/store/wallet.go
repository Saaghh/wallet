//go:build !MySql

package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Saaghh/wallet/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

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

func (p *Postgres) CreateWallet(ctx context.Context, wallet model.Wallet) (*model.Wallet, error) {
	if wallet.OwnerID == uuid.Nil {
		return nil, model.ErrNilUUID
	}

	// Checking if name is free
	query := `
	SELECT FROM wallets
	WHERE owner_id = $1 and name = $2 and is_disabled = false
`
	err := p.db.QueryRow(
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

	var pgErr *pgconn.PgError

	switch {
	case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation:
		return nil, model.ErrUserNotFound
	case err != nil:
		return nil, fmt.Errorf("p.db.QueryRow(): %w", err)
	}

	return &wallet, nil
}

func (p *Postgres) GetWallets(ctx context.Context, params model.GetParams) ([]*model.Wallet, error) {
	wallets := make([]*model.Wallet, 0, 1)

	userInfo, ok := ctx.Value(model.UserInfoKey).(model.UserInfo)
	if !ok {
		return nil, model.ErrUserInfoNotOk
	}

	query := `
	SELECT id, owner_id, currency, balance, created_at, modified_at, name
	FROM wallets
	WHERE is_disabled = false AND owner_id = $1`

	if params.Filter != "" {
		query += fmt.Sprintf(" AND name LIKE '%%%s%%'", params.Filter)
	}

	if params.Sorting != "" {
		query += " ORDER BY" + params.Sorting
		if params.Descending {
			query += " DESC"
		}
	}

	query += fmt.Sprintf(" OFFSET %d LIMIT %d", params.Offset, params.Limit)

	rows, err := p.db.Query(
		ctx,
		query,
		userInfo.ID)
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

	return wallets, nil
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

	userInfo, ok := ctx.Value(model.UserInfoKey).(model.UserInfo)
	if !ok {
		return nil, model.ErrUserInfoNotOk
	}

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, model.ErrWalletNotFound
	case err != nil:
		return nil, fmt.Errorf("p.db.QueryRow: %w", err)
	case wallet.OwnerID != userInfo.ID:
		return nil, model.ErrNotAllowed
	}

	return wallet, nil
}

func (p *Postgres) UpdateWallet(ctx context.Context, walletID uuid.UUID, request model.UpdateWalletRequest) (*model.Wallet, error) {
	wallet, err := p.GetWalletByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("p.GetWalletByID(ctx, walletID): %w", err)
	}

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
		// checking if name is free
		query := `
		SELECT FROM wallets
		WHERE owner_id = $1 and name = $2 and is_disabled = false`

		err := p.db.QueryRow(
			ctx,
			query,
			wallet.OwnerID, request.Name).Scan()

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			break
		case err != nil:
			return nil, fmt.Errorf("p.db.QueryRow(...): %w", err)
		default:
			return nil, model.ErrDuplicateWallet
		}

		query = `
		UPDATE wallets
		SET name = $2, modified_at = $3
		WHERE id = $1 AND is_disabled = false
		RETURNING id, name, modified_at
	`

		err = tx.QueryRow(
			ctx,
			query,
			walletID, request.Name, time.Now(),
		).Scan(
			&wallet.ID,
			&wallet.Name,
			&wallet.ModifiedDate)
		if err != nil {
			return nil, fmt.Errorf("p.db.QueryRow(...): %w", err)
		}
	}

	if request.Currency != nil {
		query := `
		UPDATE wallets
		SET currency = $2, modified_at = $3, balance = $4
		WHERE id = $1 AND is_disabled = false
		RETURNING id, currency, balance, modified_at
	`

		err = tx.QueryRow(
			ctx,
			query,
			walletID, request.Currency, time.Now(), wallet.Balance*request.ConversionRate,
		).Scan(
			&wallet.ID,
			&wallet.Currency,
			&wallet.Balance,
			&wallet.ModifiedDate,
		)
		if err != nil {
			return nil, fmt.Errorf("p.db.QueryRow(...): %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return wallet, nil
}

func (p *Postgres) DeleteWallet(ctx context.Context, walletID uuid.UUID) error {
	query := `
	UPDATE wallets
	SET is_disabled = true, modified_at = $2
	WHERE id = $1 AND is_disabled = false
	RETURNING id
`
	err := p.db.QueryRow(
		ctx,
		query,
		walletID, time.Now(),
	).Scan(nil)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return model.ErrWalletNotFound
	case err != nil:
		return fmt.Errorf("p.db.QueryRow(...): %w", err)
	}

	return nil
}

func (p *Postgres) Transfer(ctx context.Context, transfer model.Transfer, transaction model.Transaction) (*uuid.UUID, error) {
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
	SET balance = balance - $1, modified_at = $3
	WHERE id = $2
	RETURNING currency`

	var currency string

	err = tx.QueryRow(
		ctx,
		query,
		transfer.SumToWithdraw, transfer.AgentWallet.ID, time.Now(),
	).Scan(
		&currency,
	)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, model.ErrWalletNotFound
	case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.CheckViolation:
		return nil, model.ErrNotEnoughBalance
	case err != nil:
		return nil, fmt.Errorf("tx.QueryRow(AgentWallet).Scan(&updatedAgentWallet): %w", err)
	case currency != transfer.AgentWallet.Currency:
		return nil, model.ErrWalletWasChanged
	}

	// Depositing money

	query = `
	UPDATE wallets
	SET balance = balance + $1, modified_at = $3
	WHERE id = $2
	RETURNING currency`

	err = tx.QueryRow(
		ctx,
		query,
		transfer.SumToDeposit, transfer.TargetWallet.ID, time.Now(),
	).Scan(
		&currency,
	)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, model.ErrWalletNotFound
	case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.CheckViolation:
		return nil, model.ErrNotEnoughBalance
	case err != nil:
		return nil, fmt.Errorf("tx.QueryRow(TargetWallet).Scan(&updatedTargetWallet): %w", err)
	case currency != transfer.TargetWallet.Currency:
		return nil, model.ErrWalletWasChanged
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

	// Save transaction
	query := `
	INSERT INTO transactions (id, to_wallet_id, currency, balance)
	VALUES ($1, $2, $3, $4)
	returning id, created_at`

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

	// Update wallet
	query = `
	UPDATE wallets
	SET balance = balance + $1, modified_at = $3
	WHERE id = $2
	RETURNING currency`

	var currency string

	err = tx.QueryRow(
		ctx,
		query,
		transaction.Sum, transaction.TargetWalletID, time.Now(),
	).Scan(
		&currency)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, model.ErrWalletNotFound
	case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.CheckViolation:
		return nil, model.ErrNotEnoughBalance
	case currency != transaction.Currency:
		return nil, model.ErrWalletWasChanged
	case err != nil:
		return nil, fmt.Errorf("tx.Exec(ctx, query, targetWallet.Sum, targetWallet.ID): %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return &transaction.ID, nil
}

func (p *Postgres) GetTransactions(ctx context.Context, params model.GetParams) ([]*model.Transaction, error) {
	transactions := make([]*model.Transaction, 0, 1)

	userInfo, ok := ctx.Value(model.UserInfoKey).(model.UserInfo)
	if !ok {
		return nil, model.ErrUserInfoNotOk
	}

	query := `
	SELECT 
		transactions.id, 
		transactions.from_wallet_id, 
		transactions.to_wallet_id, 
		transactions.currency, 
		transactions.balance, 
		transactions.created_at
	FROM 
		transactions
	JOIN 
		wallets AS sender_wallet ON transactions.from_wallet_id = sender_wallet.id
	JOIN 
		wallets AS receiver_wallet ON transactions.to_wallet_id = receiver_wallet.id
	WHERE 
		sender_wallet.owner_id = $1 OR receiver_wallet.owner_id = $1`

	if params.Filter != "" {
		query += fmt.Sprintf(" AND currency LIKE '%%%s%%'", params.Filter)
	}

	if params.Sorting != "" {
		query += " ORDER BY" + params.Sorting
		if params.Descending {
			query += " DESC"
		}
	}

	query += fmt.Sprintf(" OFFSET %d LIMIT %d", params.Offset, params.Limit)

	rows, err := p.db.Query(
		ctx,
		query,
		userInfo.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("p.db.Query(ctx, query): %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		transaction := new(model.Transaction)

		err = rows.Scan(
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

	if len(transactions) == 0 {
		return nil, model.ErrTransactionsNotFound
	}

	return transactions, nil
}
