package model

import (
	"errors"
)

var (
	ErrWalletNotFound       = errors.New("wallet not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrTransactionsNotFound = errors.New("transactions not found")
	ErrWrongCurrency        = errors.New("wrong currency")
	ErrNotEnoughBalance     = errors.New("not enough balance")
	ErrNegativeSum          = errors.New("negative sum in request")
	ErrDuplicateTransaction = errors.New("transactions id already exists")
	ErrNilUUID              = errors.New("uuid is nil")
	ErrDuplicateWallet      = errors.New("duplicate wallet")
	ErrWalletWasChanged     = errors.New("wallet was changed in parallel")
	ErrZeroSum              = errors.New("sum can't be zero")
	ErrInvalidAccessToken   = errors.New("err invalid access token")
	ErrNotAllowed           = errors.New("not allowed")
	ErrUserInfoNotOk        = errors.New("user info type assertion not ok")
)
