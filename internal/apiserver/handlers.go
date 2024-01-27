package apiserver

import (
	"encoding/json"
	"fmt"
	"github.com/Saaghh/wallet/internal/model"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

func (s *APIServer) handleTime(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(time.Now().String())); err != nil {
		zap.L().With(zap.Error(err)).Warn("handleTime/w.Write(...)")

		return
	}

	s.service.SaveVisit(r.RemoteAddr)

	zap.L().Info("successful GET:/time", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) handleVisitHistory(w http.ResponseWriter, r *http.Request) {
	history, err := json.Marshal(s.service.GetVisitHistory())
	if err != nil {
		zap.L().With(zap.Error(err)).Warn("handleVisitHistory/json.Marshal(...)")

		http.Error(w, "error marshaling data", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(history); err != nil {
		zap.L().With(zap.Error(err)).Warn("handleVisitHistory/w.Write(...)")

		return
	}

	zap.L().Info("successful GET:/visitHistory", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		zap.L().With(zap.Error(err)).Info("handleUserCreate/r.ParseForm()")
		return
	}

	email := r.Form.Get("email")
	zap.L().Debug(fmt.Sprintf("handleCreateUser/email: %s", email))

	user, err := s.service.CreateUser(r.Context(), email)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		zap.L().With(zap.Error(err)).Info("handleUserCreate/s.service.CreateUser(r.Context(), email)")
		return
	}

	result, err := json.Marshal(user)
	if err != nil {
		http.Error(w, "error marshaling data", http.StatusInternalServerError)
		zap.L().With(zap.Error(err)).Warn("handleCreateUser/json.Marshal(...)")
		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(result); err != nil {
		zap.L().With(zap.Error(err)).Warn("handleCreateUser/w.Write(...)")

		return
	}

	zap.L().Info("successful POST:/user", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) handleCreateWallet(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		zap.L().With(zap.Error(err)).Info("handleCreateWallet/r.ParseForm()")
		return
	}

	currency := r.Form.Get("currency")
	ownerIdString := r.Form.Get("owner-id")

	ownerID, err := strconv.ParseInt(ownerIdString, 10, 64)
	if err != nil {
		http.Error(w, "invalid owner id", http.StatusBadRequest)
		zap.L().With(zap.Error(err)).Info("handleCreateWallet/strconv.Atoi(ownerIdString)")
		return
	}

	user := model.User{
		ID: ownerID,
	}

	wallet, err := s.service.CreateWallet(r.Context(), user, currency)
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

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(result); err != nil {
		zap.L().With(zap.Error(err)).Warn("handleCreateWallet/w.Write(...)")
		return
	}

	zap.L().Info("successful POST:/wallet", zap.String("client", r.RemoteAddr))
}

func (s *APIServer) handleGetWallet(w http.ResponseWriter, r *http.Request) {
	if !r.URL.Query().Has("wallet-id") {
		http.Error(w, "Wallet-id not found", http.StatusBadRequest)
		zap.L().Info("handleGetWallet/r.ParseForm()")
		return
	}

	walletIdString := r.URL.Query().Get("wallet-id")

	zap.L().Debug(fmt.Sprintf("handleGetWallet/walletIdString: %s", walletIdString))

	walletID, err := strconv.ParseInt(walletIdString, 10, 64)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)
		zap.L().With(zap.Error(err)).Info("handleGetWallet/strconv.Atoi(ownerIdString)")
		return
	}

	wallet, err := s.service.GetWallet(r.Context(), walletID)
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

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(result); err != nil {
		zap.L().With(zap.Error(err)).Warn("handleGetWallet/w.Write(...)")
		return
	}

	zap.L().Info("successful GET:/wallet", zap.String("client", r.RemoteAddr))

}
