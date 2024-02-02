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
	depositEndpoint  = "/wallets/deposit"
	bindAddr         = "http://localhost:8080/api/v1"
)

type IntegrationTestSuite struct {
	suite.Suite
	ctx *context.Context

	correctUser       model.User
	correctWallet     model.Wallet
	impossibleUser    model.User
	impossibleWallet  model.Wallet
	correctDeposit    model.Transaction
	incorrectDeposit  model.Transaction
	correctTransfer   model.Transaction
	incorrectTransfer model.Transaction

	wallets, transactions []int64
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.wallets = make([]int64, 0, 1)
	s.transactions = make([]int64, 0, 1)

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

	s.correctDeposit = model.Transaction{
		Currency: "USD",
		Sum:      10000,
	}

	s.correctTransfer = model.Transaction{
		Currency: "USD",
		Sum:      50,
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

// TODO check not full data for wallet create
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

	s.Run("wallet deposit / bad request", func() {
		ctx := context.Background()

		var respData model.Wallet

		resp := s.sendRequest(ctx, http.MethodPut, bindAddr+depositEndpoint, badRequestString, &respData)
		s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestPositiveScript() {
	var respWalletData model.Wallet

	wallet1 := model.Wallet{
		OwnerID:  1,
		Currency: "EUR",
	}

	wallet2 := model.Wallet{
		OwnerID:  1,
		Currency: "EUR",
	}

	ctx := context.Background()

	s.Run("creating 2 wallets", func() {
		resp := s.sendRequest(ctx, http.MethodPost, bindAddr+walletEndpoint, wallet1, &apiserver.HTTPResponse{Data: &respWalletData})
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().Equal(wallet1.Currency, respWalletData.Currency)
		s.Require().Equal(wallet1.OwnerID, respWalletData.OwnerID)
		s.Require().NotZero(respWalletData.ID)
		wallet1.ID = respWalletData.ID
		s.wallets = append(s.wallets, wallet1.ID)

		resp = s.sendRequest(ctx, http.MethodPost, bindAddr+walletEndpoint, wallet2, &apiserver.HTTPResponse{Data: &respWalletData})
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().Equal(wallet2.Currency, respWalletData.Currency)
		s.Require().Equal(wallet2.OwnerID, respWalletData.OwnerID)
		s.Require().NotZero(respWalletData.ID)
		wallet2.ID = respWalletData.ID
		s.wallets = append(s.wallets, wallet2.ID)
	})

	var transferResponse apiserver.TransferResponse

	deposit := model.Transaction{
		TargetWalletID: &wallet1.ID,
		Currency:       "EUR",
		Sum:            10000,
	}

	s.Run("depositing money to 1st wallet", func() {
		resp := s.sendRequest(ctx, http.MethodPut, bindAddr+depositEndpoint, deposit, &apiserver.HTTPResponse{Data: &transferResponse})
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().NotZero(transferResponse.TransactionID)
		s.transactions = append(s.transactions, transferResponse.TransactionID)
	})

	s.Run("checking 1st wallet", func() {
		resp := s.sendRequest(ctx, http.MethodGet, bindAddr+walletEndpoint, wallet1, &apiserver.HTTPResponse{Data: &respWalletData})
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(respWalletData.ID, wallet1.ID)
		s.Require().Equal(respWalletData.Balance, deposit.Sum)
		s.Require().Equal(respWalletData.Currency, wallet1.Currency)
	})

	transfer := model.Transaction{
		AgentWalletID:  &wallet1.ID,
		TargetWalletID: &wallet2.ID,
		Currency:       "EUR",
		Sum:            1000,
	}

	s.Run("transferring money 1 -> 2", func() {
		resp := s.sendRequest(ctx, http.MethodPut, bindAddr+transferEndpoint, transfer, &apiserver.HTTPResponse{Data: &transferResponse})
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().NotZero(transferResponse.TransactionID)
		s.transactions = append(s.transactions, transferResponse.TransactionID)
	})

	s.Run("checking both wallets", func() {
		resp := s.sendRequest(ctx, http.MethodGet, bindAddr+walletEndpoint, wallet1, &apiserver.HTTPResponse{Data: &respWalletData})
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(wallet1.ID, respWalletData.ID)
		s.Require().Equal(deposit.Sum-transfer.Sum, respWalletData.Balance)
		s.Require().Equal(wallet1.Currency, respWalletData.Currency)

		resp = s.sendRequest(ctx, http.MethodGet, bindAddr+walletEndpoint, wallet2, &apiserver.HTTPResponse{Data: &respWalletData})
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(wallet2.ID, respWalletData.ID)
		s.Require().Equal(transfer.Sum, respWalletData.Balance)
		s.Require().Equal(wallet2.Currency, respWalletData.Currency)
	})
}
