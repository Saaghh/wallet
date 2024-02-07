package model

import (
	"errors"
)

var (
	ErrWalletNotFound         = errors.New("wallet not found")
	ErrUserNotFound           = errors.New("user not found")
	ErrTransactionsNotFound   = errors.New("transactions not found")
	ErrWrongCurrency          = errors.New("wrong currency")
	ErrNotEnoughBalance       = errors.New("not enough balance")
	ErrNegativeRequestBalance = errors.New("negative balance in request")
	ErrDuplicateTransaction   = errors.New("transactions id already exists")
	ErrNilUUID                = errors.New("uuid is nil")
	ErrDuplicateWallet        = errors.New("duplicate wallet")
	ErrWalletWasChanged       = errors.New("wallet was changed in parallel")
)
