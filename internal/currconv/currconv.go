package currconv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Saaghh/wallet/internal/apiserver"
	"github.com/Saaghh/wallet/internal/model"
	"go.uber.org/zap"
)

type metrics interface {
	TrackExternalRequest(start time.Time, endpoint string)
}

type RemoteCurrencyConverter struct {
	XRAddress string
	metrics   metrics
}

const xrEndpoint string = "/xr"

func New(xrBindAddr string, metrics metrics) *RemoteCurrencyConverter {
	return &RemoteCurrencyConverter{
		XRAddress: "http://localhost" + xrBindAddr + xrEndpoint,
		metrics:   metrics,
	}
}

func (c *RemoteCurrencyConverter) GetExchangeRate(baseCurrency, targetCurrency string) (float64, error) {
	defer c.metrics.TrackExternalRequest(time.Now(), c.XRAddress)

	queryParams := fmt.Sprintf("?base=%s&target=%s", baseCurrency, targetCurrency)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.XRAddress+queryParams,
		nil)
	if err != nil {
		return 0, fmt.Errorf("server.NewRequestWithContext(...): %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			zap.L().With(zap.Error(err)).Warn("GetExchangeRate/resp.Body.Close()")
		}
	}()

	switch resp.StatusCode {
	case http.StatusBadRequest:
		return 0, model.ErrWrongCurrency
	case http.StatusInternalServerError:
		return 0, model.ErrGettingXR
	}

	var xrResponse model.XRResponse

	err = json.NewDecoder(resp.Body).Decode(&apiserver.HTTPResponse{Data: &xrResponse})
	if err != nil {
		return 0, fmt.Errorf("json.NewDecoder(resp.Body).Decode(...): %w", err)
	}

	return xrResponse.XR, nil
}
