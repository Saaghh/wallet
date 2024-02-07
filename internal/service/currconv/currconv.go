package currconv

import "github.com/Saaghh/wallet/internal/model"

type MockConverter struct{}

func New() *MockConverter {
	return &MockConverter{}
}

func (c *MockConverter) GetExchangeRate(baseCurrency, targetCurrency string) (float64, error) {
	if !(baseCurrency == "USD" || baseCurrency == "EUR" || baseCurrency == "RUB") {
		return 0, model.ErrWrongCurrency
	}

	if !(targetCurrency == "USD" || targetCurrency == "EUR" || targetCurrency == "RUB") {
		return 0, model.ErrWrongCurrency
	}

	return 1, nil
}
