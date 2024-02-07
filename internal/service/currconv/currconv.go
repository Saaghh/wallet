package currconv

import "github.com/Saaghh/wallet/internal/model"

type MockConverter struct {
	currencies map[string]float64
}

func New() *MockConverter {
	currencies := map[string]float64{
		"RUB": 1,
		"USD": 90.53,
		"EUR": 97.53,
		"KZT": 20.0115,
		"IDR": 0.00579328,
	}

	return &MockConverter{
		currencies: currencies,
	}
}

func (c *MockConverter) GetExchangeRate(baseCurrency, targetCurrency string) (float64, error) {

	baseK, ok := c.currencies[baseCurrency]
	if !ok {
		return 0, model.ErrWrongCurrency
	}

	targetK, ok := c.currencies[targetCurrency]
	if !ok {
		return 0, model.ErrWrongCurrency
	}

	return baseK / targetK, nil
}
