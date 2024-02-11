package service

import (
	"context"
	"fmt"

	"github.com/Saaghh/wallet/internal/model"
	"github.com/google/uuid"
)

type store interface {
	CreateWallet(ctx context.Context, wallet model.Wallet) (*model.Wallet, error)
	GetWalletByID(ctx context.Context, walletID uuid.UUID) (*model.Wallet, error)
	GetWallets(ctx context.Context, params model.GetParams) ([]*model.Wallet, error)
	DeleteWallet(ctx context.Context, walletID uuid.UUID) error
	UpdateWallet(ctx context.Context, walletID uuid.UUID, request model.UpdateWalletRequest) (*model.Wallet, error)

	GetTransactions(ctx context.Context, params model.GetParams) ([]*model.Transaction, error)
	Transfer(ctx context.Context, transfer model.Transfer, transaction model.Transaction) (*uuid.UUID, error)
	ExternalTransaction(ctx context.Context, transaction model.Transaction) (*uuid.UUID, error)
}

type currencyConverter interface {
	GetExchangeRate(baseCurrency, targetCurrency string) (float64, error)
}

type Service struct {
	db store
	cc currencyConverter
}

func New(db store, cc currencyConverter) *Service {
	return &Service{
		db: db,
		cc: cc,
	}
}

func (s *Service) CreateWallet(ctx context.Context, wallet model.Wallet) (*model.Wallet, error) {
	rWallet, err := s.db.CreateWallet(ctx, wallet)
	if err != nil {
		return nil, fmt.Errorf("s.db.CreateWallet(ctx, owner, currency): %w", err)
	}

	return rWallet, nil
}

func (s *Service) GetWalletByID(ctx context.Context, walletID uuid.UUID) (*model.Wallet, error) {
	wallet, err := s.db.GetWalletByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWalletByID(ctx, walletID): %w", err)
	}

	return wallet, nil
}

func (s *Service) transactionToTransfer(ctx context.Context, transaction model.Transaction) (*model.Transfer, error) {
	agentWallet, err := s.db.GetWalletByID(ctx, *transaction.AgentWalletID)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWalletByID(ctx, *transaction.AgentWalletID): %w", err)
	}

	targetWallet, err := s.db.GetWalletByID(ctx, *transaction.TargetWalletID)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWalletByID(ctx, *transaction.TargetWallet): %w", err)
	}

	transfer := model.Transfer{
		ID:           transaction.ID,
		AgentWallet:  agentWallet,
		TargetWallet: targetWallet,
	}

	if agentWallet.Currency != transaction.Currency {
		xr, err := s.cc.GetExchangeRate(transaction.Currency, agentWallet.Currency)
		if err != nil {
			return nil, fmt.Errorf("s.cc.GetExchangeRate(transaction.Currency, agentWallet.Currency): %w", err)
		}

		transfer.SumToWithdraw = xr * transaction.Sum
	} else {
		transfer.SumToWithdraw = transaction.Sum
	}

	if targetWallet.Currency != transaction.Currency {
		xr, err := s.cc.GetExchangeRate(transaction.Currency, agentWallet.Currency)
		if err != nil {
			return nil, fmt.Errorf("s.cc.GetExchangeRate(transaction.Currency, agentWallet.Currency): %w", err)
		}

		transfer.SumToDeposit = xr * transaction.Sum
	} else {
		transfer.SumToDeposit = transaction.Sum
	}

	return &transfer, nil
}

func (s *Service) Transfer(ctx context.Context, transaction model.Transaction) (*uuid.UUID, error) {
	// conversion
	transfer, err := s.transactionToTransfer(ctx, transaction)
	if err != nil {
		return nil, fmt.Errorf("s.transactionToTransfer(ctx, transaction): %w", err)
	}

	// execution
	transactionID, err := s.db.Transfer(ctx, *transfer, transaction)
	if err != nil {
		return nil, fmt.Errorf("s.db.Transfer(ctx, transaction): %w", err)
	}

	return transactionID, nil
}

func (s *Service) ExternalTransaction(ctx context.Context, transaction model.Transaction) (*uuid.UUID, error) {
	// conversion
	wallet, err := s.db.GetWalletByID(ctx, *transaction.TargetWalletID)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWalletByID(ctx, *transaction.TargetWalletID): %w", err)
	}

	if wallet.Currency != transaction.Currency {
		k, err := s.cc.GetExchangeRate(transaction.Currency, wallet.Currency)
		if err != nil {
			return nil, fmt.Errorf("s.cc.GetExchangeRate(transaction.Currency, wallet.Currency): %w", err)
		}

		transaction.Currency = wallet.Currency
		transaction.Sum *= k
	}

	// execution
	transactionID, err := s.db.ExternalTransaction(ctx, transaction)
	if err != nil {
		return nil, fmt.Errorf("s.db.ExternalTransaction(ctx, transaction): %w", err)
	}

	return transactionID, nil
}

func (s *Service) GetWallets(ctx context.Context, params model.GetParams) ([]*model.Wallet, error) {
	wallets, err := s.db.GetWallets(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWallets(ctx, owner): %w", err)
	}

	return wallets, nil
}

func (s *Service) DeleteWallet(ctx context.Context, walletID uuid.UUID) error {
	err := s.db.DeleteWallet(ctx, walletID)
	if err != nil {
		return fmt.Errorf("s.db.DeleteWallet(ctx, walletID): %w", err)
	}

	return nil
}

func (s *Service) UpdateWallet(ctx context.Context, walletID uuid.UUID, request model.UpdateWalletRequest) (*model.Wallet, error) {
	wallet, err := s.db.GetWalletByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWalletByID(ctx, walletID): %w", err)
	}

	userInfo := ctx.Value(model.UserInfoKey).(model.UserInfo)

	if userInfo.ID != wallet.OwnerID {
		return nil, model.ErrNotAllowed
	}

	if request.Currency != nil && *request.Currency != wallet.Currency {
		xr, err := s.cc.GetExchangeRate(wallet.Currency, *request.Currency)
		if err != nil {
			return nil, fmt.Errorf("s.cc.GetExchangeRate(*request.Currency, wallet.Currency): %w", err)
		}

		request.ConversionRate = xr
	} else {
		request.ConversionRate = 1
	}

	wallet, err = s.db.UpdateWallet(ctx, walletID, request)
	if err != nil {
		return nil, fmt.Errorf("s.db.UpdateWallet(ctx, walletID, request): %w", err)
	}

	return wallet, nil
}

func (s *Service) GetTransactions(ctx context.Context, params model.GetParams) ([]*model.Transaction, error) {
	transactions, err := s.db.GetTransactions(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetTransactions(ctx): %w", err)
	}

	return transactions, nil
}
