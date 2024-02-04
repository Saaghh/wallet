package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"

	"github.com/Saaghh/wallet/internal/model"
	"go.uber.org/zap"
)

type HTTPResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type TransferResponse struct {
	TransactionID int64 `json:"transactionId"`
}

type service interface {
	CreateWallet(ctx context.Context, wallet model.Wallet) (*model.Wallet, error)
	GetWalletByID(ctx context.Context, walletID int64) (*model.Wallet, error)
	GetWallets(ctx context.Context) ([]*model.Wallet, error)
	DeleteWallet(ctx context.Context, walletID int64) error
	UpdateWallet(ctx context.Context, walletID int64, request model.UpdateWalletRequest) (*model.Wallet, error)

	GetTransactions(ctx context.Context) ([]*model.Transaction, error)
	Transfer(ctx context.Context, wtx model.Transaction) (int64, error)
	ExternalTransaction(ctx context.Context, transaction model.Transaction) (int64, error)
}

func (s *APIServer) createWallet(w http.ResponseWriter, r *http.Request) {
	var rWallet model.Wallet

	if err := json.NewDecoder(r.Body).Decode(&rWallet); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "failed to read body")
		return
	}

	wallet, err := s.service.CreateWallet(r.Context(), rWallet)

	switch {
	case errors.Is(err, model.ErrUserNotFound):
		writeErrorResponse(w, http.StatusNotFound, "user not found")
		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn(
			"createWallet/s.service.CreateWallet(r.Context(), model.User{ID: rWallet.OwnerID}, rWallet.Currency)")
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeOkResponse(w, http.StatusCreated, wallet)

	zap.L().Debug("successful POST:/wallet", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) getWallets(w http.ResponseWriter, r *http.Request) {
	wallets, err := s.service.GetWallets(r.Context())
	switch {
	case errors.Is(err, model.ErrWalletNotFound):
		writeErrorResponse(w, http.StatusNotFound, "wallets not found")
		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("getWallets/s.service.GetWallets(r.Context(), rUser)")
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeOkResponse(w, http.StatusOK, wallets)

	zap.L().Debug("successful GET:/wallets", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) getWalletByID(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "can't get id")
		return
	}

	wallet, err := s.service.GetWalletByID(r.Context(), id)
	switch {
	case errors.Is(err, model.ErrWalletNotFound):
		writeErrorResponse(w, http.StatusNotFound, "wallet not found")
		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("getWalletByID/s.service.GetWalletByID(r.Context(), walletID))")
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeOkResponse(w, http.StatusOK, wallet)

	zap.L().Debug("successful GET:/wallets/{id}", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) updateWallet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "can't get id")
		return
	}

	var updateRequest model.UpdateWalletRequest

	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "failed to read body")
		return
	}

	wallet, err := s.service.UpdateWallet(r.Context(), id, updateRequest)
	switch {
	case errors.Is(err, model.ErrWalletNotFound):
		writeErrorResponse(w, http.StatusNotFound, "wallet not found")
		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn(
			"updateWallet/s.service.UpdateWallet(r.Context(), id, updateRequest)")
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeOkResponse(w, http.StatusOK, wallet)

	zap.L().Debug("successful PATCH:/wallets/{id}", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) deleteWallet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "can't get id")
		return
	}

	err = s.service.DeleteWallet(r.Context(), id)
	switch {
	case errors.Is(err, model.ErrWalletNotFound):
		writeErrorResponse(w, http.StatusNotFound, "wallet not found")
		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("deleteWallet/s.service.DeleteWallet(r.Context(), id)")
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)

	zap.L().Debug("successful DELETE:/wallets/{id}", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) deposit(w http.ResponseWriter, r *http.Request) {
	var requestTransaction model.Transaction

	err := json.NewDecoder(r.Body).Decode(&requestTransaction)
	switch {
	case err != nil:
		writeErrorResponse(w, http.StatusBadRequest, "failed to read body")
		return
	case requestTransaction.Sum == 0:
		writeErrorResponse(w, http.StatusUnprocessableEntity, "sum can't be zero")
		return
	case requestTransaction.Sum < 0:
		writeErrorResponse(w, http.StatusUnprocessableEntity, "sum must be positive")
		return
	case requestTransaction.TargetWalletID == nil:
		writeErrorResponse(w, http.StatusUnprocessableEntity, "target wallet not found")
		return
	}

	transferID, err := s.service.ExternalTransaction(r.Context(), requestTransaction)
	switch {
	case errors.Is(err, model.ErrWalletNotFound):
		writeErrorResponse(w, http.StatusNotFound, "wallet not found")
		return
	case errors.Is(err, model.ErrWrongCurrency):
		fallthrough
	case errors.Is(err, model.ErrNegativeRequestBalance):
		writeErrorResponse(w, http.StatusUnprocessableEntity, "incorrect request data")
		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("deposit/s.service.ExternalTransaction(r.Context(), requestTransaction)")
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeOkResponse(w, http.StatusOK, TransferResponse{TransactionID: transferID})

	zap.L().Debug("successful PUT:/deposit", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) transfer(w http.ResponseWriter, r *http.Request) {
	var requestTransaction model.Transaction

	if err := json.NewDecoder(r.Body).Decode(&requestTransaction); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "failed to read body")
		return
	}

	transferID, err := s.service.Transfer(r.Context(), requestTransaction)
	switch {
	case errors.Is(err, model.ErrWalletNotFound):
		writeErrorResponse(w, http.StatusNotFound, "wallet not found")
		return
	case errors.Is(err, model.ErrNotEnoughBalance):
		fallthrough
	case errors.Is(err, model.ErrWrongCurrency):
		fallthrough
	case errors.Is(err, model.ErrNegativeRequestBalance):
		writeErrorResponse(w, http.StatusUnprocessableEntity, "incorrect request data")
		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("s.service.Transfer(r.Context(), requestTransaction)")
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeOkResponse(w, http.StatusOK, TransferResponse{TransactionID: transferID})

	zap.L().Debug("successful PUT:/transfer", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) withdraw(w http.ResponseWriter, r *http.Request) {
	var requestTransaction model.Transaction

	err := json.NewDecoder(r.Body).Decode(&requestTransaction)
	switch {
	case err != nil:
		writeErrorResponse(w, http.StatusBadRequest, "failed to read body")
		return
	case requestTransaction.Sum <= 0:
		writeErrorResponse(w, http.StatusUnprocessableEntity, "sum must be >0")
		return
	}

	requestTransaction.Sum *= -1

	transferID, err := s.service.ExternalTransaction(r.Context(), requestTransaction)
	switch {
	case errors.Is(err, model.ErrWalletNotFound):
		writeErrorResponse(w, http.StatusNotFound, "wallet not found")
		return
	case errors.Is(err, model.ErrWrongCurrency):
		writeErrorResponse(w, http.StatusUnprocessableEntity, "incorrect request data")
		return
	case errors.Is(err, model.ErrNotEnoughBalance):
		writeErrorResponse(w, http.StatusUnprocessableEntity, "not enough balance")
		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("withdraw/s.service.ExternalTransaction(r.Context(), requestTransaction)")
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeOkResponse(w, http.StatusOK, TransferResponse{TransactionID: transferID})

	zap.L().Debug("successful PUT:/withdraw", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) getTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := s.service.GetTransactions(r.Context())
	switch {
	case errors.Is(err, model.ErrTransactionsNotFound):
		writeErrorResponse(w, http.StatusNotFound, "transactions not found")
		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("getTransactions/s.service.GetTransactions(r.Context())")
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeOkResponse(w, http.StatusOK, transactions)

	zap.L().Debug("successful GET:/wallets/transactions", zap.String("client", r.RemoteAddr))
}

func writeOkResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	err := json.NewEncoder(w).Encode(HTTPResponse{Data: data})
	if err != nil {
		zap.L().With(zap.Error(err)).Warn(
			"writeOkResponse/json.NewEncoder(w).Encode(HTTPResponse{Data: data})")
	}

}

func writeErrorResponse(w http.ResponseWriter, statusCode int, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	err := json.NewEncoder(w).Encode(HTTPResponse{Error: description})
	if err != nil {
		zap.L().With(zap.Error(err)).Warn(
			"writeErrorResponse/json.NewEncoder(w).Encode(HTTPResponse{Error: data})")
	}
}
