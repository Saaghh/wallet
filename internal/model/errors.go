package model

import (
	"errors"
	"net/http"
)

var (
	ErrWalletNotFound         = errors.New("wallet not found")
	ErrUserNotFound           = errors.New("user not found")
	ErrWrongCurrency          = errors.New("wrong currency")
	ErrNotEnoughBalance       = errors.New("not enough balance")
	ErrNegativeRequestBalance = errors.New("negative balance in request")
	ErrInternalServerError    = errors.New(http.StatusText(http.StatusInternalServerError))
)
