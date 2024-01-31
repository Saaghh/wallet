package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/Saaghh/wallet/internal/apiserver"
	"github.com/Saaghh/wallet/internal/config"
	"github.com/Saaghh/wallet/internal/logger"
	"github.com/Saaghh/wallet/internal/model"
	"github.com/Saaghh/wallet/internal/service"
	"github.com/Saaghh/wallet/internal/store"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/suite"
)

const (
	walletEndpoint   = "/wallets"
	transferEndpoint = "/wallets/transfer"
	bindAddr         = "http://localhost:8080/api/v1"
)

type IntegrationTestSuite struct {
	suite.Suite
	ctx *context.Context

	correctUser      model.User
	correctWallet    model.Wallet
	impossibleUser   model.User
	impossibleWallet model.Wallet
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	s.ctx = &ctx

	cfg := config.New()

	logger.InitLogger(logger.Config{Level: "Warn"})

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

	s.impossibleUser = model.User{
		ID:      -1,
		Email:   "impossible@example.com",
		RegDate: time.Now(),
	}

	s.impossibleWallet = model.Wallet{
		ID:      -1,
		OwnerID: -1,
	}
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

func (s *IntegrationTestSuite) TestPositiveWallet() {
	s.Run("wallet creation", func() {
		ctx := context.Background()

		var respData model.Wallet

		resp := s.sendRequest(ctx, http.MethodPost, bindAddr+walletEndpoint, s.correctWallet, &apiserver.HTTPResponse{Data: &respData})
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().Equal(s.correctWallet.Currency, respData.Currency)
		s.Require().Equal(s.correctWallet.OwnerID, respData.OwnerID)
		s.Require().NotZero(respData.ID)
		s.correctWallet = respData
	})

	s.Run("wallet request", func() {
		ctx := context.Background()

		var respData model.Wallet

		resp := s.sendRequest(ctx, http.MethodGet, bindAddr+walletEndpoint, s.correctWallet, &apiserver.HTTPResponse{Data: &respData})
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.correctWallet, respData)
	})
}

func (s *IntegrationTestSuite) TestNegativeWallet() {
	s.Run("wallet creation / user not found", func() {
		ctx := context.Background()

		var respData apiserver.HTTPResponse

		resp := s.sendRequest(ctx, http.MethodPost, bindAddr+walletEndpoint, s.impossibleWallet, &respData)
		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("wallet request / wallet not found", func() {
		ctx := context.Background()

		var respData apiserver.HTTPResponse

		resp := s.sendRequest(ctx, http.MethodGet, bindAddr+walletEndpoint, s.impossibleWallet, &respData)
		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestBadRequests() {
	badRequestString := "Lorem ipsum dolor sit amet"

	s.Run("wallet creation / bad request", func() {
		ctx := context.Background()

		var respData model.Wallet

		resp := s.sendRequest(ctx, http.MethodPost, bindAddr+walletEndpoint, badRequestString, &respData)
		s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	})

	s.Run("wallet request / bad request", func() {
		ctx := context.Background()

		var respData model.Wallet

		resp := s.sendRequest(ctx, http.MethodGet, bindAddr+walletEndpoint, badRequestString, &respData)
		s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	})

	s.Run("wallet transfer / bad request", func() {
		ctx := context.Background()

		var respData model.Wallet

		resp := s.sendRequest(ctx, http.MethodPut, bindAddr+transferEndpoint, badRequestString, &respData)
		s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	})
}
