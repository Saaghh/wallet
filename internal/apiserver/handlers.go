package apiserver

import (
	"encoding/json"
	"net/http"
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

	zap.L().Info("sent /time", zap.String("client", r.RemoteAddr))
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

	zap.L().Info("sent /visitHistory", zap.String("client", r.RemoteAddr))
}
