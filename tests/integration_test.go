package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Saaghh/wallet/internal/apiserver"
	"github.com/Saaghh/wallet/internal/config"
	"github.com/Saaghh/wallet/internal/logger"
	"github.com/Saaghh/wallet/internal/model"
	"github.com/Saaghh/wallet/internal/service"
	"github.com/Saaghh/wallet/internal/store"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

const (
	walletEndpoint = "/wallet"
	userEndpoint   = "/user"
	bindAddr       = "http://localhost:8080"
)

type IntegrationTestSuite struct {
	suite.Suite
	ctx *context.Context
	str *store.Postgres

	correctUser   model.User
	correctWallet model.Wallet
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	s.ctx = &ctx

	cfg := config.New()

	logger.InitLogger(logger.Config{Level: cfg.LogLevel})

	str, err := store.New(ctx, cfg)
	s.Require().NoError(err)

	err = str.Migrate(migrate.Up)
	s.Require().NoError(err)

	srv := service.New(str)

	server := apiserver.New(apiserver.Config{BindAddress: cfg.BindAddress}, srv)

	go func() {
		err = server.Run(ctx)
		s.Require().NoError(err)
	}()
}

func (s *IntegrationTestSuite) SetupTest() {
	s.correctUser = model.User{
		ID:      1,
		Email:   "123@example.com",
		RegDate: time.Now(),
	}

	s.correctWallet = model.Wallet{
		OwnerID:  1,
		Currency: "USD",
	}
}

func (s *IntegrationTestSuite) TeardownSuite() {

}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) sendRequest(ctx context.Context, method, endpoint string, body interface{}, dest interface{}) *http.Response {
	s.T().Helper()

	reqBody, err := json.Marshal(body)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(reqBody))
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)

	defer func() {
		err = resp.Body.Close()
		s.Require().NoError(err)
	}()

	if dest != nil {
		err = json.NewDecoder(resp.Body).Decode(&dest)
		s.Require().NoError(err)
	}

	return resp
}

func (s *IntegrationTestSuite) TestUserCreation() {

	s.Run("normal wallet creation", func() {
		ctx := context.Background()

		var respData model.Wallet

		resp := s.sendRequest(ctx, http.MethodPost, bindAddr+walletEndpoint, s.correctWallet, &respData)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().Equal(s.correctWallet.Currency, respData.Currency)
		s.Require().Equal(s.correctWallet.OwnerID, respData.OwnerID)
		s.Require().NotZero(respData.ID)
		s.correctWallet.ID = respData.ID
	})

	s.Run("normal wallet request", func() {
		ctx := context.Background()

		var respData model.Wallet

		resp := s.sendRequest(ctx, http.MethodGet, bindAddr+walletEndpoint, s.correctWallet, &respData)
		s.Require().Equal(http.StatusOK, resp.StatusCode)

	})

}
