package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

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
	CreateWallet(ctx context.Context, owner model.User, currency string) (*model.Wallet, error)
	GetWallet(ctx context.Context, walletID int64) (*model.Wallet, error)
	Transfer(ctx context.Context, wtx model.Transaction) (int64, error)
	Deposit(ctx context.Context, transaction model.Transaction) (int64, error)
}

func (s *APIServer) handleCreateWallet(w http.ResponseWriter, r *http.Request) {
	var rWallet model.Wallet

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewDecoder(r.Body).Decode(&rWallet); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		err = json.NewEncoder(w).Encode(HTTPResponse{Error: "failed to read body"})

		if err != nil {
			zap.L().With(zap.Error(err)).Warn(
				"handleCreateWallet/json.NewEncoder(w).Encode(HTTPResponse{Error: \"failed to read body\"})")
		}

		return
	}

	wallet, err := s.service.CreateWallet(r.Context(), model.User{ID: rWallet.OwnerID}, rWallet.Currency)

	switch {
	case errors.Is(err, model.ErrUserNotFound):
		w.WriteHeader(http.StatusNotFound)
		err = json.NewEncoder(w).Encode(HTTPResponse{Error: err.Error()})

		if err != nil {
			zap.L().With(zap.Error(err)).Warn(
				"handleCreateWallet/json.NewEncoder(w).Encode(HTTPResponse{Error: \"failed to read body\"})")
		}

		return
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(w).Encode(HTTPResponse{Error: "internal server error"})

		if err != nil {
			zap.L().With(zap.Error(err)).Warn(
				"handleCreateWallet/json.NewEncoder(w).Encode(HTTPResponse{Error: \"failed to read body\"})")
		}

		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(HTTPResponse{Data: wallet})

	if err != nil {
		zap.L().With(zap.Error(err)).Warn("handleCreateWallet/json.NewEncoder(w).Encode(HTTPResponse{Data: wallet}}")

		return
	}

	zap.L().Debug("successful POST:/wallet", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) handleGetWallet(w http.ResponseWriter, r *http.Request) {
	var rWallet model.Wallet

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewDecoder(r.Body).Decode(&rWallet); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		err = json.NewEncoder(w).Encode(HTTPResponse{Error: "failed to read body"})

		if err != nil {
			zap.L().With(zap.Error(err)).Warn(
				"handleGetWallet/json.NewEncoder(w).Encode(HTTPResponse{Error: \"failed to read body\"})")
		}

		return
	}

	wallet, err := s.service.GetWallet(r.Context(), rWallet.ID)

	switch {
	case errors.Is(err, model.ErrWalletNotFound):
		w.WriteHeader(http.StatusNotFound)
		err = json.NewEncoder(w).Encode(HTTPResponse{Error: err.Error()})

		if err != nil {
			zap.L().With(zap.Error(err)).Warn(
				"handleGetWallet/json.NewEncoder(w).Encode(HTTPResponse{Error: err.Error()})")
		}

		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("handleGetWallet/s.service.GetWallet(r.Context(), walletID))")
		w.WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(w).Encode(HTTPResponse{Error: "internal server error"})

		if err != nil {
			zap.L().With(zap.Error(err)).Warn(
				"handleGetWallet/json.NewEncoder(w).Encode(HTTPResponse{Error: \"internal server error\"})")
		}

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(HTTPResponse{Data: wallet})
	if err != nil {
		zap.L().With(zap.Error(err)).Warn(
			"handleGetWallet/json.NewEncoder(w).Encode(HTTPResponse{Data: wallet}}")

		return
	}

	zap.L().Debug("successful GET:/wallet", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) {
	var requestTransaction model.Transaction

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewDecoder(r.Body).Decode(&requestTransaction); err != nil {
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(HTTPResponse{Error: "failed to read body"})
		if err != nil {
			zap.L().With(zap.Error(err)).Warn(
				"handleTransfer/json.NewEncoder(w).Encode(HTTPResponse{Error: \"failed to read body\"})")
		}

		return
	}

	transferID, err := s.service.Transfer(r.Context(), requestTransaction)
	if err != nil {
		status := http.StatusInternalServerError

		switch {
		case errors.Is(err, model.ErrWalletNotFound):
			status = http.StatusNotFound
		case errors.Is(err, model.ErrNotEnoughBalance):
			fallthrough
		case errors.Is(err, model.ErrWrongCurrency):
			fallthrough
		case errors.Is(err, model.ErrNegativeRequestBalance):
			status = http.StatusUnprocessableEntity

		default:
			zap.L().With(zap.Error(err)).Warn("s.service.Transfer(r.Context(), requestTransaction)")
			err = model.ErrInternalServerError
		}

		w.WriteHeader(status)
		err = json.NewEncoder(w).Encode(HTTPResponse{Error: err.Error()})

		if err != nil {
			zap.L().With(zap.Error(err)).Warn("json.NewEncoder(w).Encode(HTTPResponse{Error: err.Error()})")
		}

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(HTTPResponse{Data: TransferResponse{TransactionID: transferID}})
	if err != nil {
		zap.L().With(zap.Error(err)).Warn(
			"handleTransfer/json.NewEncoder(w).Encode(HTTPResponse{Data: TransferResponse{TransactionID: transferID}}")

		return
	}

	zap.L().Debug("successful PUT:/transfer", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) handleDeposit(w http.ResponseWriter, r *http.Request) {
	var requestTransaction model.Transaction

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewDecoder(r.Body).Decode(&requestTransaction); err != nil {
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(HTTPResponse{Error: "failed to read body"})
		if err != nil {
			zap.L().With(zap.Error(err)).Warn(
				"handeDeposit/json.NewEncoder(w).Encode(HTTPResponse{Error: \"failed to read body\"})")
		}

		return
	}

	transferID, err := s.service.Deposit(r.Context(), requestTransaction)
	if err != nil {
		status := http.StatusInternalServerError

		switch {
		case errors.Is(err, model.ErrWalletNotFound):
			status = http.StatusNotFound
		case errors.Is(err, model.ErrWrongCurrency):
			fallthrough
		case errors.Is(err, model.ErrNegativeRequestBalance):
			status = http.StatusUnprocessableEntity

		default:
			zap.L().With(zap.Error(err)).Warn("handleDeposit/s.service.Deposit(r.Context(), requestTransaction)")
			err = model.ErrInternalServerError
		}

		w.WriteHeader(status)

		err = json.NewEncoder(w).Encode(HTTPResponse{Error: err.Error()})
		if err != nil {
			zap.L().With(zap.Error(err)).Warn("handleDeposit/json.NewEncoder(w).Encode(HTTPResponse{Error: err.Error()})")
		}

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(HTTPResponse{Data: TransferResponse{TransactionID: transferID}})
	if err != nil {
		zap.L().With(zap.Error(err)).Warn(
			"handleDeposit/json.NewEncoder(w).Encode(HTTPResponse{Data: TransferResponse{TransactionID: transferID}}")

		return
	}

	zap.L().Debug("successful PUT:/deposit", zap.String("client", r.RemoteAddr))
}
