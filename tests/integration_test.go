package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"testing"

	"github.com/Saaghh/wallet/internal/apiserver"
	"github.com/Saaghh/wallet/internal/config"
	"github.com/Saaghh/wallet/internal/currconv"
	"github.com/Saaghh/wallet/internal/jwtgenerator"
	"github.com/Saaghh/wallet/internal/logger"
	"github.com/Saaghh/wallet/internal/model"
	"github.com/Saaghh/wallet/internal/prometrics"
	"github.com/Saaghh/wallet/internal/service"
	"github.com/Saaghh/wallet/internal/store"
	"github.com/google/uuid"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const (
	walletEndpoint       = "/wallets"
	transferEndpoint     = "/wallets/transfer"
	depositEndpoint      = "/wallets/deposit"
	withdrawEndpoint     = "/wallets/withdraw"
	transactionsEndpoint = "/wallets/transactions"
	bindAddr             = "http://localhost:8080/api/v1"
	currencyEUR          = "EUR"
	currencyUSD          = "USD"
	standardName         = "good wallet"
	secondaryName        = "better wallet"
	thirdName            = "best wallet"
	fourthName           = "fourth name wallet"
	badRequestString     = "Lorem Ipsum?"
)

type currencyConverter interface {
	GetExchangeRate(baseCurrency, targetCurrency string) (float64, error)
}

type IntegrationTestSuite struct {
	suite.Suite
	ctx *context.Context

	testOwnerID   uuid.UUID
	secondOwnerID uuid.UUID

	str *store.Postgres

	converter currencyConverter

	authToken       string
	secondAuthToken string

	tokenGenerator *jwtgenerator.JWTGenerator
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.testOwnerID = uuid.New()

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	s.ctx = &ctx

	cfg := config.New()

	logger.InitLogger(logger.Config{Level: "Warn"})

	str, err := store.New(ctx, cfg)
	s.Require().NoError(err)

	err = str.Migrate(migrate.Up)
	s.Require().NoError(err)

	s.tokenGenerator = jwtgenerator.NewJWTGenerator()

	user, err := str.CreateUser(ctx, model.User{Email: "test@test.com"})
	s.Require().NoError(err)
	s.testOwnerID = user.ID
	s.authToken, err = s.tokenGenerator.GetNewTokenString(*user)
	s.Require().NoError(err)

	user, err = str.CreateUser(ctx, model.User{Email: "test2@test.com"})
	s.Require().NoError(err)
	s.secondOwnerID = user.ID
	s.secondAuthToken, err = s.tokenGenerator.GetNewTokenString(*user)
	s.Require().NoError(err)

	s.str = str

	metrics := prometrics.New()

	s.converter = currconv.New(cfg.XRBindAddr, metrics)

	srv := service.New(str, s.converter)

	server := apiserver.New(apiserver.Config{BindAddress: cfg.BindAddress}, srv, s.tokenGenerator.GetPublicKey(), metrics)

	go func() {
		err = server.Run(ctx)
		s.Require().NoError(err)
	}()
}

func (s *IntegrationTestSuite) TearDownSuite() {
	err := s.str.TruncateTables(context.Background())
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TestWallets() {
	wallet1 := model.Wallet{
		OwnerID:  s.testOwnerID,
		Currency: currencyEUR,
		Name:     standardName,
		Balance:  0,
	}

	wallet2 := model.Wallet{
		OwnerID:  s.testOwnerID,
		Currency: currencyEUR,
		Name:     secondaryName,
		Balance:  0,
	}

	wallet3 := model.Wallet{
		OwnerID:  s.testOwnerID,
		Currency: currencyUSD,
		Name:     thirdName,
		Balance:  0,
	}

	s.Run("401", func() {
		temp := s.authToken
		s.authToken = ""

		resp := s.sendRequest(
			context.Background(),
			http.MethodGet,
			walletEndpoint,
			nil,
			nil)

		s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)

		s.authToken = temp
	})

	s.Run("GET:/wallets/empty", func() {
		wallets := make([]model.Wallet, 0)

		resp := s.sendRequest(
			context.Background(),
			http.MethodGet,
			walletEndpoint,
			nil,
			&apiserver.HTTPResponse{Data: &wallets})

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Zero(len(wallets))
	})

	s.Run("GET:/transactions", func() {
		resp := s.sendRequest(
			context.Background(),
			http.MethodGet,
			transactionsEndpoint,
			nil,
			nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("wallets", func() {
		s.Run("POST:/wallets", func() {
			s.Run("201", func() {
				s.checkWalletPost(&wallet1)
				s.checkWalletPost(&wallet2)
				s.checkWalletPost(&wallet3)
			})

			s.Run("422/duplicate name", func() {
				var respWalletData model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodPost,
					walletEndpoint,
					wallet1,
					&apiserver.HTTPResponse{Data: &respWalletData})
				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("400", func() {
				resp := s.sendRequest(
					context.Background(),
					http.MethodPost,
					walletEndpoint,
					badRequestString,
					nil)
				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})
		})

		s.Run("GET:/wallets", func() {
			s.Run("200", func() {
				var wallets []model.Wallet

				params := "?limit=10&sorting=created_at&descending=true"

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+params,
					nil,
					&apiserver.HTTPResponse{Data: &wallets})

				walletsFound := 0

				for _, value := range wallets {
					if value.ID == wallet1.ID || value.ID == wallet2.ID || value.ID == wallet3.ID {
						walletsFound++
					}
				}

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().NotZero(len(wallets))
				s.Require().Equal(3, walletsFound)
			})

			s.Run("200/empty array", func() {
				var wallets []model.Wallet

				temp := s.authToken
				s.authToken = s.secondAuthToken
				defer func() { s.authToken = temp }()

				params := "?limit=10&sorting=created_at&descending=true"

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+params,
					nil,
					&apiserver.HTTPResponse{Data: &wallets})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Zero(len(wallets))
			})
		})
	})

	s.Run("wallets/{id}", func() {
		s.Run("GET:/wallets", func() {
			s.Run("200", func() {
				var respData model.Wallet

				resp := s.sendRequest(context.Background(), http.MethodGet, walletEndpoint+"/"+wallet1.ID.String(), nil, &apiserver.HTTPResponse{Data: &respData})
				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(wallet1.OwnerID, respData.OwnerID)
				s.Require().Equal(wallet1.Currency, respData.Currency)
				s.Require().Equal(wallet1.Name, respData.Name)
				s.Require().Equal(wallet1.Balance, respData.Balance)
			})

			s.Run("400", func() {
				resp := s.sendRequest(context.Background(), http.MethodGet, walletEndpoint+"/"+badRequestString, nil, nil)
				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("404/not allowed user", func() {
				var respData model.Wallet

				temp := s.authToken
				s.authToken = s.secondAuthToken

				resp := s.sendRequest(context.Background(), http.MethodGet, walletEndpoint+"/"+wallet1.ID.String(), nil, &apiserver.HTTPResponse{Data: &respData})
				s.Require().Equal(http.StatusNotFound, resp.StatusCode)

				s.authToken = temp
			})

			s.Run("404", func() {
				resp := s.sendRequest(context.Background(), http.MethodGet, walletEndpoint+"/"+uuid.New().String(), nil, nil)
				s.Require().Equal(http.StatusNotFound, resp.StatusCode)
			})
		})

		s.Run("PATCH:/wallets", func() {
			s.Run("200/name", func() {
				var respData model.Wallet

				newName := fourthName
				wallet := &wallet1

				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					walletEndpoint+"/"+wallet.ID.String(),
					model.UpdateWalletRequest{Name: &newName},
					&apiserver.HTTPResponse{Data: &respData})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(newName, respData.Name)
				s.Require().Equal(wallet.ID, respData.ID)

				wallet.Name = newName
			})

			s.Run("deposit some money", func() {
				wallet := &wallet1

				trasaction := model.Transaction{
					ID:             uuid.New(),
					TargetWalletID: &wallet.ID,
					Currency:       wallet.Currency,
					Sum:            100,
				}

				var respData apiserver.TransferResponse

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					depositEndpoint,
					trasaction,
					&apiserver.HTTPResponse{Data: &respData})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().NotZero(respData.TransactionID)
				wallet1.Balance += trasaction.Sum
			})

			s.Run("200/currency", func() {
				var respData model.Wallet

				newCurrency := currencyUSD
				wallet := &wallet1
				xr, err := s.converter.GetExchangeRate(wallet.Currency, newCurrency)
				s.Require().NoError(err)

				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					walletEndpoint+"/"+wallet.ID.String(),
					model.UpdateWalletRequest{Currency: &newCurrency},
					&apiserver.HTTPResponse{Data: &respData})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(newCurrency, respData.Currency)
				s.Require().Equal(wallet.ID, respData.ID)
				s.Require().Equal(wallet.Balance*xr, respData.Balance)

				wallet.Currency = newCurrency
				wallet.Balance *= xr
			})

			s.Run("200/both", func() {
				var respData model.Wallet

				newCurrency := currencyUSD
				newName := standardName
				wallet := &wallet2

				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					walletEndpoint+"/"+wallet.ID.String(),
					model.UpdateWalletRequest{Currency: &newCurrency, Name: &newName},
					&apiserver.HTTPResponse{Data: &respData})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(newCurrency, respData.Currency)
				s.Require().Equal(newName, respData.Name)
				s.Require().Equal(wallet.ID, respData.ID)

				wallet.Currency = newCurrency
				wallet.Name = newName
			})

			s.Run("401", func() {
				var respData model.Wallet

				newCurrency := currencyUSD
				newName := standardName
				wallet := &wallet2

				temp := s.authToken

				s.authToken = s.secondAuthToken

				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					walletEndpoint+"/"+wallet.ID.String(),
					model.UpdateWalletRequest{Currency: &newCurrency, Name: &newName},
					&apiserver.HTTPResponse{Data: &respData})

				s.authToken = temp

				s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
			})

			s.Run("422", func() {
				impossibleCurrency := "Non existent"

				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					walletEndpoint+"/"+wallet2.ID.String(),
					model.UpdateWalletRequest{Currency: &impossibleCurrency},
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("400/id", func() {
				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					walletEndpoint+"/"+badRequestString,
					nil,
					nil)
				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("400/body", func() {
				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					walletEndpoint+"/"+wallet1.ID.String(),
					badRequestString,
					nil)
				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("404", func() {
				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					walletEndpoint+"/"+uuid.Nil.String(),
					nil,
					nil)
				s.Require().Equal(http.StatusNotFound, resp.StatusCode)
			})
		})

		s.Run("DELETE:/wallets", func() {
			s.Run("400", func() {
				resp := s.sendRequest(
					context.Background(),
					http.MethodDelete,
					walletEndpoint+"/"+badRequestString,
					nil,
					nil)
				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("404", func() {
				resp := s.sendRequest(
					context.Background(),
					http.MethodDelete,
					walletEndpoint+"/"+uuid.Nil.String(),
					nil,
					nil)
				s.Require().Equal(http.StatusNotFound, resp.StatusCode)
			})

			s.Run("204", func() {
				wallet := &wallet3

				resp := s.sendRequest(
					context.Background(),
					http.MethodDelete,
					walletEndpoint+"/"+wallet.ID.String(),
					nil,
					nil)

				s.Require().Equal(http.StatusNoContent, resp.StatusCode)

				s.Run("404/get deleted wallet", func() {
					resp := s.sendRequest(
						context.Background(),
						http.MethodGet,
						walletEndpoint+"/"+wallet.ID.String(),
						nil,
						nil)

					s.Require().Equal(http.StatusNotFound, resp.StatusCode)
				})

				s.Run("no deleted wallet in full list", func() {
					var wallets []model.Wallet

					params := "?limit=10"

					resp := s.sendRequest(
						context.Background(),
						http.MethodGet,
						walletEndpoint+params,
						nil,
						&apiserver.HTTPResponse{Data: &wallets})

					s.Require().Equal(http.StatusOK, resp.StatusCode)

					isWalletFound := false

					for _, value := range wallets {
						if value.ID == wallet.ID {
							isWalletFound = true
						}
					}

					s.Require().False(isWalletFound)
				})
			})
		})
	})

	s.Run("wallets/deposit", func() {
		s.Run("400", func() {
			// TODO BUG: panic with empty body?
			resp := s.sendRequest(context.Background(), http.MethodPut, depositEndpoint, badRequestString, nil)
			s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
		})

		s.Run("404", func() {
			trans := model.Transaction{
				ID:            uuid.Must(uuid.NewRandom()),
				AgentWalletID: &wallet1.ID,
				Currency:      currencyUSD,
				Sum:           1000,
			}

			iWalletID := uuid.Nil
			var respData apiserver.HTTPResponse
			trans.TargetWalletID = &iWalletID

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				depositEndpoint,
				trans,
				&respData)

			s.Require().Equal(http.StatusNotFound, resp.StatusCode)
		})

		s.Run("422", func() {
			s.Run("negative sum", func() {
				trans := model.Transaction{
					ID:             uuid.New(),
					TargetWalletID: &wallet1.ID,
					Currency:       currencyUSD,
					Sum:            -1,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					depositEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("zero sum", func() {
				trans := model.Transaction{
					ID:             uuid.New(),
					TargetWalletID: &wallet1.ID,
					Currency:       currencyUSD,
					Sum:            0,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					depositEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("wrong currency", func() {
				trans := model.Transaction{
					ID:             uuid.New(),
					TargetWalletID: &wallet1.ID,
					Currency:       "impossible currency",
					Sum:            1000,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					depositEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})
		})

		trans := model.Transaction{
			ID:             uuid.New(),
			TargetWalletID: &wallet1.ID,
			Currency:       currencyUSD,
			Sum:            1000,
		}

		s.Run("200", func() {
			var transferResponse apiserver.TransferResponse

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				depositEndpoint,
				trans,
				&apiserver.HTTPResponse{Data: &transferResponse})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(transferResponse.TransactionID)
			trans.ID = transferResponse.TransactionID
			wallet1.Balance += trans.Sum
		})

		s.Run("200/another currency", func() {
			var transferResponse apiserver.TransferResponse

			trans := model.Transaction{
				ID:             uuid.New(),
				TargetWalletID: &wallet1.ID,
				Currency:       "IDR",
				Sum:            10000,
			}

			xr, err := s.converter.GetExchangeRate(trans.Currency, wallet1.Currency)
			s.Require().NoError(err)

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				depositEndpoint,
				trans,
				&apiserver.HTTPResponse{Data: &transferResponse})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(transferResponse.TransactionID)

			s.Run("check wallet balance", func() {
				var wallet model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+"/"+wallet1.ID.String(),
					nil,
					&apiserver.HTTPResponse{Data: &wallet})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(wallet.ID, wallet.ID)
				s.Require().Equal(wallet1.Balance+trans.Sum*xr, wallet.Balance)
				wallet1 = wallet
			})
		})

		s.Run("429", func() {
			var transferResponse apiserver.TransferResponse

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				depositEndpoint,
				trans,
				&apiserver.HTTPResponse{Data: &transferResponse})

			s.Require().Equal(http.StatusTooManyRequests, resp.StatusCode)
		})
	})

	s.Run("wallets/transfer", func() {
		s.Run("400", func() {
			resp := s.sendRequest(context.Background(), http.MethodPut, transferEndpoint, badRequestString, nil)
			s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
		})

		s.Run("404", func() {
			s.Run("agent wallet not found", func() {
				impWID := uuid.Nil

				trans := model.Transaction{
					ID:             uuid.New(),
					AgentWalletID:  &impWID,
					TargetWalletID: &wallet2.ID,
					Currency:       currencyUSD,
					Sum:            300,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					transferEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusNotFound, resp.StatusCode)
			})

			s.Run("target wallet not found", func() {
				impWID := uuid.Nil

				trans := model.Transaction{
					ID:             uuid.New(),
					AgentWalletID:  &wallet1.ID,
					TargetWalletID: &impWID,
					Currency:       currencyUSD,
					Sum:            300,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					transferEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusNotFound, resp.StatusCode)
			})
		})

		s.Run("422", func() {
			s.Run("not enough balance", func() {
				trans := model.Transaction{
					ID:             uuid.New(),
					AgentWalletID:  &wallet1.ID,
					TargetWalletID: &wallet2.ID,
					Currency:       currencyUSD,
					Sum:            2000,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					transferEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("negative sum", func() {
				trans := model.Transaction{
					ID:             uuid.New(),
					AgentWalletID:  &wallet1.ID,
					TargetWalletID: &wallet2.ID,
					Currency:       currencyEUR,
					Sum:            -300,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					transferEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("wrong currency", func() {
				trans := model.Transaction{
					ID:             uuid.New(),
					AgentWalletID:  &wallet1.ID,
					TargetWalletID: &wallet2.ID,
					Currency:       "impossible currency",
					Sum:            1000,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					transferEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})
		})

		trans := model.Transaction{
			ID:             uuid.New(),
			AgentWalletID:  &wallet1.ID,
			TargetWalletID: &wallet2.ID,
			Currency:       currencyUSD,
			Sum:            300,
		}

		s.Run("200", func() {
			var respData apiserver.TransferResponse

			id := trans.ID.String()
			zap.L().Debug(id)

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				transferEndpoint,
				trans,
				&apiserver.HTTPResponse{Data: &respData})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(respData.TransactionID)
			wallet1.Balance -= trans.Sum
			wallet2.Balance += trans.Sum
		})

		s.Run("429", func() {
			var respData apiserver.HTTPResponse

			id := trans.ID.String()
			zap.L().Debug(id)

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				transferEndpoint,
				trans,
				&respData)

			s.Require().Equal(http.StatusTooManyRequests, resp.StatusCode)
		})

		s.Run("200/another currency", func() {
			trans.ID = uuid.New()
			trans.Currency = "KZT"
			trans.Sum = 2.5

			var respData apiserver.TransferResponse

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				transferEndpoint,
				trans,
				&apiserver.HTTPResponse{Data: &respData})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(respData.TransactionID)

			s.Run("check first wallet", func() {
				xr, err := s.converter.GetExchangeRate(trans.Currency, wallet1.Currency)
				s.Require().NoError(err)

				wallet := s.getWalletByID(wallet1.ID)

				s.Require().Equal(wallet1.Balance-trans.Sum*xr, wallet.Balance)

				wallet1 = *wallet
			})

			s.Run("check second wallet", func() {
				xr, err := s.converter.GetExchangeRate(trans.Currency, wallet2.Currency)
				s.Require().NoError(err)

				wallet := s.getWalletByID(wallet2.ID)

				s.Require().Equal(wallet2.Balance+trans.Sum*xr, wallet.Balance)

				wallet2 = *wallet
			})
		})
	})

	s.Run("wallets/withdraw", func() {
		s.Run("400", func() {
			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				withdrawEndpoint,
				badRequestString,
				nil)
			s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
		})

		s.Run("404", func() {
			walletID := uuid.Nil
			trans := model.Transaction{
				ID:             uuid.New(),
				TargetWalletID: &walletID,
				Currency:       currencyUSD,
				Sum:            300,
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				withdrawEndpoint,
				trans,
				nil)

			s.Require().Equal(http.StatusNotFound, resp.StatusCode)
		})

		s.Run("422", func() {
			s.Run("negative sum", func() {
				trans := model.Transaction{
					TargetWalletID: &wallet2.ID,
					Currency:       currencyUSD,
					Sum:            -100,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					withdrawEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("zero sum", func() {
				trans := model.Transaction{
					TargetWalletID: &wallet2.ID,
					Currency:       currencyUSD,
					Sum:            0,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					withdrawEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("not enough balance", func() {
				trans := model.Transaction{
					ID:             uuid.New(),
					TargetWalletID: &wallet2.ID,
					Currency:       currencyUSD,
					Sum:            3000,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					withdrawEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("wrong currency", func() {
				trans := model.Transaction{
					ID:             uuid.New(),
					TargetWalletID: &wallet2.ID,
					Currency:       "impossible currency",
					Sum:            10,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					withdrawEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})
		})

		trans := model.Transaction{
			ID:             uuid.New(),
			TargetWalletID: &wallet2.ID,
			Currency:       currencyUSD,
			Sum:            100,
		}

		s.Run("200", func() {
			var transferResponse apiserver.TransferResponse

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				withdrawEndpoint,
				trans,
				&apiserver.HTTPResponse{Data: &transferResponse})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(transferResponse.TransactionID)
			wallet2.Balance -= trans.Sum
		})

		s.Run("429", func() {
			var respData apiserver.HTTPResponse

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				withdrawEndpoint,
				trans,
				&respData)

			s.Require().Equal(http.StatusTooManyRequests, resp.StatusCode)
		})

		s.Run("200/another currency", func() {
			trans.Sum = 10
			trans.Currency = "IDR"
			trans.ID = uuid.New()

			var transferResponse apiserver.TransferResponse

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				withdrawEndpoint,
				trans,
				&apiserver.HTTPResponse{Data: &transferResponse})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(transferResponse.TransactionID)

			s.Run("check wallet", func() {
				xr, err := s.converter.GetExchangeRate(trans.Currency, wallet2.Currency)
				s.Require().NoError(err)

				wallet := s.getWalletByID(wallet2.ID)

				s.Require().Equal(wallet2.Balance-trans.Sum*xr, wallet.Balance)
			})
		})
	})

	s.Run("wallets/transactions", func() {
		s.Run("200", func() {
			var transactions []model.Transaction

			params := "?limit=10&sorting=created_at&descending=true"

			resp := s.sendRequest(
				context.Background(),
				http.MethodGet,
				transactionsEndpoint+params,
				nil,
				&apiserver.HTTPResponse{Data: &transactions})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(len(transactions))
		})

		s.Run("404", func() {
			var transactions []model.Transaction

			temp := s.authToken
			s.authToken = s.secondAuthToken
			defer func() { s.authToken = temp }()

			params := "?limit=10"

			resp := s.sendRequest(
				context.Background(),
				http.MethodGet,
				transactionsEndpoint+params,
				nil,
				&apiserver.HTTPResponse{Data: &transactions})

			s.Require().Equal(http.StatusNotFound, resp.StatusCode)
		})
	})

	s.Run("listing", func() {
		// create 29 wallets
		// test paging
		// test filtration
		// test sorting
		// test combos

		// setup test cycle
		temp := s.authToken
		s.authToken = s.secondAuthToken
		defer func() { s.authToken = temp }()

		s.Run("create 29 wallets", func() {
			for i := 0; i < 29; i++ {
				wallet := model.Wallet{
					Currency: currencyUSD,
					Name:     strconv.Itoa(i),
					OwnerID:  s.secondOwnerID,
				}
				s.checkWalletPost(&wallet)
			}
		})

		s.Run("check pages", func() {
			limit := 10
			offset := 0

			s.Run("page 1", func() {
				query := fmt.Sprintf("?limit=%d&offset=%d", limit, offset)

				var page []model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+query,
					nil,
					&apiserver.HTTPResponse{Data: &page})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(limit, len(page))
			})

			offset += 10

			s.Run("page 2", func() {
				query := fmt.Sprintf("?limit=%d&offset=%d", limit, offset)

				var page []model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+query,
					nil,
					&apiserver.HTTPResponse{Data: &page})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(limit, len(page))
			})

			offset += 10

			s.Run("page 3", func() {
				query := fmt.Sprintf("?limit=%d&offset=%d", limit, offset)

				var page []model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+query,
					nil,
					&apiserver.HTTPResponse{Data: &page})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Greater(limit, len(page))
			})

			offset += 10

			s.Run("page 4/404", func() {
				query := fmt.Sprintf("?limit=%d&offset=%d", limit, offset)

				var page []model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+query,
					nil,
					&apiserver.HTTPResponse{Data: &page})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Zero(len(page))
			})
		})

		s.Run("check filters for 1", func() {
			s.Run("1 in name", func() {
				limit := 20
				offset := 0
				// filtering works only by name
				filter := "1"
				const walletsWithFilterInName int = 12

				query := fmt.Sprintf("?limit=%d&offset=%d&filter=%s", limit, offset, filter)

				var page []model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+query,
					nil,
					&apiserver.HTTPResponse{Data: &page})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(walletsWithFilterInName, len(page))
				for i := 0; i < len(page); i++ {
					s.Require().Contains(page[i].Name, filter)
				}
			})

			s.Run("0 in name", func() {
				limit := 20
				offset := 0
				// filtering works only by name
				filter := "0"
				const walletsWithFilterInName int = 3

				query := fmt.Sprintf("?limit=%d&offset=%d&filter=%s", limit, offset, filter)

				var page []model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+query,
					nil,
					&apiserver.HTTPResponse{Data: &page})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(walletsWithFilterInName, len(page))
				for i := 0; i < len(page); i++ {
					s.Require().Contains(page[i].Name, filter)
				}
			})

			s.Run("letter a in name/404", func() {
				limit := 20
				offset := 0
				// filtering works only by name
				filter := "a"

				query := fmt.Sprintf("?limit=%d&offset=%d&filter=%s", limit, offset, filter)

				var page []model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+query,
					nil,
					&apiserver.HTTPResponse{Data: &page})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Zero(len(page))
			})
		})

		s.Run("check sorting", func() {
			s.Run("created_at, desc", func() {
				sorting := "created_at"
				descending := "true"

				query := fmt.Sprintf("?sorting=%s&descending=%s", sorting, descending)

				var page []model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+query,
					nil,
					&apiserver.HTTPResponse{Data: &page})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				for i := 1; i < len(page); i++ {
					s.Require().True(page[i-1].CreatedDate.After(page[i].CreatedDate))
				}
			})

			s.Run("name, asc", func() {
				limit := 30
				sorting := "name"
				descending := "false"

				query := fmt.Sprintf("?sorting=%s&descending=%s&limit=%d", sorting, descending, limit)

				var page []model.Wallet

				resp := s.sendRequest(
					context.Background(),
					http.MethodGet,
					walletEndpoint+query,
					nil,
					&apiserver.HTTPResponse{Data: &page})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				// filling expected order
				order := make([]string, 0)
				for i := 0; i < len(page); i++ {
					order = append(order, strconv.Itoa(i))
				}
				sort.Strings(order)

				// checking if equal
				for i := 0; i < len(page); i++ {
					s.Require().Equal(order[i], page[i].Name)
				}
			})
		})
	})

	s.Run("archive", func() {
		wallets, err := s.str.DisableInactiveWallets(context.Background())
		s.Require().NoError(err)
		s.Require().Equal(0, len(wallets))
	})
}

func (s *IntegrationTestSuite) checkWalletPost(wallet *model.Wallet) {
	var respWalletData model.Wallet

	resp := s.sendRequest(context.Background(), http.MethodPost, walletEndpoint, wallet, &apiserver.HTTPResponse{Data: &respWalletData})
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().Equal(wallet.Currency, respWalletData.Currency)
	s.Require().Equal(wallet.OwnerID, respWalletData.OwnerID)
	s.Require().Equal(wallet.Balance, respWalletData.Balance)
	s.Require().Equal(wallet.Name, respWalletData.Name)
	s.Require().NotZero(respWalletData.ID)
	wallet.ID = respWalletData.ID
}

func (s *IntegrationTestSuite) getWalletByID(id uuid.UUID) *model.Wallet {
	var wallet model.Wallet

	resp := s.sendRequest(
		context.Background(),
		http.MethodGet,
		walletEndpoint+"/"+id.String(),
		nil,
		&apiserver.HTTPResponse{Data: &wallet})

	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().Equal(id, wallet.ID)

	return &wallet
}

func (s *IntegrationTestSuite) sendRequest(ctx context.Context, method, endpoint string, body interface{}, dest interface{}) *http.Response {
	s.T().Helper()

	reqBody, err := json.Marshal(body)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(ctx, method, bindAddr+endpoint, bytes.NewReader(reqBody))
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Authorization", "Bearer "+s.authToken)

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
