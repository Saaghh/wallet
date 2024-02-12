package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Saaghh/wallet/internal/model"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type HTTPResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func (s *Server) handleGetExchangeRate(w http.ResponseWriter, r *http.Request) {
	xrRequest, err := valuesToXRRequest(r.URL.Query())
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "error getting params")

		return
	}

	xr, err := s.getExchangeRate(xrRequest.BaseCurrency, xrRequest.TargetCurrency)

	switch {
	case errors.Is(err, model.ErrWrongCurrency):
		writeErrorResponse(w, http.StatusBadRequest, "wrong currency")

		return
	case err != nil:
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")

		return
	}

	zap.L().Debug(
		"successful GET:/xr",
		zap.Float64("xr", xr),
		zap.String("base", xrRequest.BaseCurrency),
		zap.String("target", xrRequest.TargetCurrency))

	writeOkResponse(w, http.StatusOK, model.XRResponse{XR: xr})
}

func (s *Server) getExchangeRate(baseCurrency, targetCurrency string) (float64, error) {
	baseK, ok := s.currencies[baseCurrency]
	if !ok {
		return 0, model.ErrWrongCurrency
	}

	targetK, ok := s.currencies[targetCurrency]
	if !ok {
		return 0, model.ErrWrongCurrency
	}

	return baseK / targetK, nil
}

func valuesToXRRequest(values url.Values) (*model.XRRequest, error) {
	decoder := schema.NewDecoder()

	xrRequest := &model.XRRequest{}

	err := decoder.Decode(xrRequest, values)
	if err != nil {
		return nil, fmt.Errorf("decoder.Decode(xrRequest, values): %w", err)
	}

	return xrRequest, nil
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
