package apiserver

import (
	"encoding/json"
	"github.com/Saaghh/wallet/internal/model"
	"go.uber.org/zap"
	"net/http"
)

func (s *APIServer) handleCreateWallet(w http.ResponseWriter, r *http.Request) {
	var rWallet model.Wallet

	if err := json.NewDecoder(r.Body).Decode(&rWallet); err != nil {
		http.Error(w, "failed read body", http.StatusBadRequest)
		zap.L().With(zap.Error(err)).Info("handleCreateWallet/json.NewDecoder(r.Body).Decode(&rWallet)")
		return
	}

	user := model.User{
		ID: rWallet.OwnerID,
	}

	wallet, err := s.service.CreateWallet(r.Context(), user, rWallet.Currency)
	if err != nil {
		http.Error(w, "Failed to create wallet", http.StatusInternalServerError)
		zap.L().With(zap.Error(err)).Info("handleCreateWallet/s.service.CreateWallet(r.Context(), user, currency)")
		return
	}

	result, err := json.Marshal(wallet)
	if err != nil {
		http.Error(w, "error marshaling data", http.StatusInternalServerError)
		zap.L().With(zap.Error(err)).Warn("handleCreateWallet/json.Marshal(...)")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(result); err != nil {
		zap.L().With(zap.Error(err)).Warn("handleCreateWallet/w.Write(...)")
		return
	}

	zap.L().Info("successful POST:/wallet", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) handleGetWallet(w http.ResponseWriter, r *http.Request) {
	var rWallet model.Wallet

	if err := json.NewDecoder(r.Body).Decode(&rWallet); err != nil {
		http.Error(w, "failed read body", http.StatusBadRequest)
		zap.L().With(zap.Error(err)).Info("handleGetWallet/json.NewDecoder(r.Body).Decode(&rWallet)")
		return
	}

	wallet, err := s.service.GetWallet(r.Context(), rWallet.ID)
	if err != nil {
		http.Error(w, "Failed to get wallet", http.StatusInternalServerError)
		zap.L().With(zap.Error(err)).Info("handleGetWallet/s.service.GetWallet(r.Context(), walletID))")
		return
	}

	result, err := json.Marshal(wallet)
	if err != nil {
		http.Error(w, "error marshaling data", http.StatusInternalServerError)
		zap.L().With(zap.Error(err)).Warn("handleGetWallet/json.Marshal(...)")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(result); err != nil {
		zap.L().With(zap.Error(err)).Warn("handleGetWallet/w.Write(...)")
		return
	}

	zap.L().Info("successful GET:/wallet", zap.String("client", r.RemoteAddr))

}
