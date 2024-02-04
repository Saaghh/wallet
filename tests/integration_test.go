package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os/signal"
	"strconv"
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
	walletEndpoint       = "/wallets"
	transferEndpoint     = "/wallets/transfer"
	depositEndpoint      = "/wallets/deposit"
	withdrawEndpoint     = "/wallets/withdraw"
	transactionsEndpoint = "/wallets/transactions"
	bindAddr             = "http://localhost:8080/api/v1"
	currencyEUR          = "EUR"
	currencyUSD          = "USD"
	prescriptedOwnerID   = 1
	impossibleOwnerID    = -1
	impossibleWalletID   = -1
	standartName         = "good wallet"
	secondayName         = "better wallet"
	thirdName            = "best wallet"
	badRequestString     = "Lorem Ipsum?"
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

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
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

func (s *IntegrationTestSuite) TestWallets() {

	wallet1 := model.Wallet{
		OwnerID:  prescriptedOwnerID,
		Currency: currencyEUR,
		Name:     standartName,
	}

	wallet2 := model.Wallet{
		OwnerID:  prescriptedOwnerID,
		Currency: currencyEUR,
		Name:     standartName,
	}

	wallet3 := model.Wallet{
		OwnerID:  prescriptedOwnerID,
		Currency: currencyUSD,
		Name:     standartName,
	}

	s.Run("/wallets", func() {
		s.Run("POST:/wallets", func() {
			s.Run("201", func() {
				s.checkWalletPost(&wallet1)
				s.checkWalletPost(&wallet2)
				s.checkWalletPost(&wallet3)
			})

			s.Run("400", func() {
				resp := s.sendRequest(context.Background(), http.MethodPost, bindAddr+walletEndpoint, badRequestString, nil)
				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("404", func() {
				resp := s.sendRequest(context.Background(), http.MethodPost, bindAddr+walletEndpoint, model.Wallet{
					OwnerID:  impossibleOwnerID,
					Currency: currencyEUR,
					Name:     standartName,
				}, nil)
				s.Require().Equal(http.StatusNotFound, resp.StatusCode)
			})
		})

		s.Run("GET:/wallets", func() {
			s.Run("200", func() {
				var wallets []model.Wallet
				resp := s.sendRequest(context.Background(), http.MethodGet, bindAddr+walletEndpoint, nil, &apiserver.HTTPResponse{Data: &wallets})

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

			s.Run("404", func() {
				//TODO get 404

				s.Require().True(true)
			})
		})
	})

	s.Run("/wallets/{id}", func() {
		s.Run("GET:/wallets", func() {
			s.Run("200", func() {
				var respData model.Wallet

				resp := s.sendRequest(context.Background(), http.MethodGet, bindAddr+walletEndpoint+"/"+strconv.FormatInt(wallet1.ID, 10), nil, &apiserver.HTTPResponse{Data: &respData})
				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(wallet1.OwnerID, respData.OwnerID)
				s.Require().Equal(wallet1.Currency, respData.Currency)
				s.Require().Equal(wallet1.Name, respData.Name)
				s.Require().Equal(wallet1.Balance, respData.Balance)
			})

			s.Run("400", func() {
				resp := s.sendRequest(context.Background(), http.MethodGet, bindAddr+walletEndpoint+"/"+badRequestString, nil, nil)
				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("404", func() {
				resp := s.sendRequest(context.Background(), http.MethodGet, bindAddr+walletEndpoint+"/-1", nil, nil)
				s.Require().Equal(http.StatusNotFound, resp.StatusCode)
			})
		})

		s.Run("PATCH:/wallets", func() {
			s.Run("200/name", func() {
				var respData model.Wallet

				newName := secondayName
				wallet := &wallet1

				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					bindAddr+walletEndpoint+"/"+strconv.FormatInt(wallet.ID, 10),
					model.UpdateWalletRequest{Name: &newName},
					&apiserver.HTTPResponse{Data: &respData})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(newName, respData.Name)
				s.Require().Equal(wallet.ID, respData.ID)

				wallet.Name = newName
			})

			s.Run("200/currency", func() {
				var respData model.Wallet

				newCurrency := currencyUSD
				wallet := &wallet1

				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					bindAddr+walletEndpoint+"/"+strconv.FormatInt(wallet.ID, 10),
					model.UpdateWalletRequest{Currency: &newCurrency},
					&apiserver.HTTPResponse{Data: &respData})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(newCurrency, respData.Currency)
				s.Require().Equal(wallet.ID, respData.ID)

				wallet.Currency = newCurrency
			})

			s.Run("200/both", func() {
				var respData model.Wallet

				newCurrency := currencyUSD
				newName := secondayName
				wallet := &wallet2

				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					bindAddr+walletEndpoint+"/"+strconv.FormatInt(wallet.ID, 10),
					model.UpdateWalletRequest{Currency: &newCurrency, Name: &newName},
					&apiserver.HTTPResponse{Data: &respData})

				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().Equal(newCurrency, respData.Currency)
				s.Require().Equal(newName, respData.Name)
				s.Require().Equal(wallet.ID, respData.ID)

				wallet.Currency = newCurrency
				wallet.Name = newName
			})

			s.Run("400/id", func() {
				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					bindAddr+walletEndpoint+"/"+badRequestString,
					nil,
					nil)
				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("400/body", func() {
				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					bindAddr+walletEndpoint+"/"+strconv.FormatInt(wallet1.ID, 10),
					badRequestString,
					nil)
				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("404", func() {
				resp := s.sendRequest(
					context.Background(),
					http.MethodPatch,
					bindAddr+walletEndpoint+"/"+strconv.FormatInt(impossibleWalletID, 10),
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
					bindAddr+walletEndpoint+"/"+badRequestString,
					nil,
					nil)
				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("404", func() {
				resp := s.sendRequest(
					context.Background(),
					http.MethodDelete,
					bindAddr+walletEndpoint+"/"+strconv.FormatInt(impossibleWalletID, 10),
					nil,
					nil)
				s.Require().Equal(http.StatusNotFound, resp.StatusCode)
			})

			s.Run("204", func() {
				wallet := &wallet3

				resp := s.sendRequest(
					context.Background(),
					http.MethodDelete,
					bindAddr+walletEndpoint+"/"+strconv.FormatInt(wallet.ID, 10),
					nil,
					nil)

				s.Require().Equal(http.StatusNoContent, resp.StatusCode)

				s.Run("404/get deleted wallet", func() {
					resp := s.sendRequest(
						context.Background(),
						http.MethodGet,
						bindAddr+walletEndpoint+"/"+strconv.FormatInt(wallet.ID, 10),
						nil,
						nil)

					s.Require().Equal(http.StatusNotFound, resp.StatusCode)
				})

				s.Run("no deleted wallet in full list", func() {
					var wallets []model.Wallet
					resp := s.sendRequest(
						context.Background(),
						http.MethodGet,
						bindAddr+walletEndpoint,
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

	s.Run("/wallets/deposit", func() {
		s.Run("400", func() {
			//TODO BUG: panic with empty body?
			resp := s.sendRequest(context.Background(), http.MethodPut, bindAddr+depositEndpoint, badRequestString, nil)
			s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
		})

		trans := model.Transaction{
			AgentWalletID: &wallet1.ID,
			Currency:      currencyUSD,
			Sum:           1000,
		}

		s.Run("404", func() {

			var iWalletID int64 = impossibleWalletID

			trans.TargetWalletID = &iWalletID

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				bindAddr+depositEndpoint,
				trans,
				nil)

			s.Require().Equal(http.StatusNotFound, resp.StatusCode)

		})

		s.Run("422", func() {
			s.Run("negative sum", func() {
				trans = model.Transaction{
					TargetWalletID: &wallet1.ID,
					Currency:       currencyUSD,
					Sum:            -1,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					bindAddr+depositEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("wrong currency", func() {
				trans = model.Transaction{
					TargetWalletID: &wallet1.ID,
					Currency:       currencyEUR,
					Sum:            1000,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					bindAddr+depositEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("zero sum", func() {
				trans = model.Transaction{
					TargetWalletID: &wallet1.ID,
					Currency:       currencyUSD,
					Sum:            0,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					bindAddr+depositEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})
		})

		s.Run("200", func() {
			trans = model.Transaction{
				TargetWalletID: &wallet1.ID,
				Currency:       currencyUSD,
				Sum:            1000,
			}

			var transferResponse apiserver.TransferResponse

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				bindAddr+depositEndpoint,
				trans,
				&apiserver.HTTPResponse{Data: &transferResponse})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(transferResponse.TransactionID)
			trans.ID = transferResponse.TransactionID
			s.transactions = append(s.transactions, trans.ID)
		})
	})

	s.Run("/wallets/transfer", func() {
		s.Run("400", func() {
			resp := s.sendRequest(context.Background(), http.MethodPut, bindAddr+transferEndpoint, badRequestString, nil)
			s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
		})

		s.Run("404", func() {
			s.Run("agent wallet not found", func() {

				var impWID int64 = impossibleWalletID

				trans := model.Transaction{
					AgentWalletID:  &impWID,
					TargetWalletID: &wallet2.ID,
					Currency:       currencyUSD,
					Sum:            300,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					bindAddr+transferEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusNotFound, resp.StatusCode)
			})

			s.Run("target wallet not found", func() {
				var impWID int64 = impossibleWalletID

				trans := model.Transaction{
					AgentWalletID:  &wallet1.ID,
					TargetWalletID: &impWID,
					Currency:       currencyUSD,
					Sum:            300,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					bindAddr+transferEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusNotFound, resp.StatusCode)
			})
		})

		s.Run("422", func() {
			s.Run("not enough balance", func() {
				trans := model.Transaction{
					AgentWalletID:  &wallet1.ID,
					TargetWalletID: &wallet2.ID,
					Currency:       currencyUSD,
					Sum:            2000,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					bindAddr+transferEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("wrong currency", func() {
				trans := model.Transaction{
					AgentWalletID:  &wallet1.ID,
					TargetWalletID: &wallet2.ID,
					Currency:       currencyEUR,
					Sum:            300,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					bindAddr+transferEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("negative sum", func() {
				trans := model.Transaction{
					AgentWalletID:  &wallet1.ID,
					TargetWalletID: &wallet2.ID,
					Currency:       currencyEUR,
					Sum:            -300,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					bindAddr+transferEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})
		})

		s.Run("200", func() {

			var respData apiserver.TransferResponse

			trans := model.Transaction{
				AgentWalletID:  &wallet1.ID,
				TargetWalletID: &wallet2.ID,
				Currency:       currencyUSD,
				Sum:            300,
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				bindAddr+transferEndpoint,
				trans,
				&apiserver.HTTPResponse{Data: &respData})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(respData.TransactionID)
			s.transactions = append(s.transactions, respData.TransactionID)
		})
	})

	s.Run("/wallets/withdraw", func() {
		s.Run("400", func() {
			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				bindAddr+withdrawEndpoint,
				badRequestString,
				nil)
			s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
		})

		s.Run("404", func() {

			var walletID int64 = impossibleWalletID
			trans := model.Transaction{
				TargetWalletID: &walletID,
				Currency:       currencyUSD,
				Sum:            300,
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				bindAddr+withdrawEndpoint,
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
					bindAddr+withdrawEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("wrong currency", func() {
				trans := model.Transaction{
					TargetWalletID: &wallet2.ID,
					Currency:       currencyEUR,
					Sum:            300,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					bindAddr+depositEndpoint,
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
					bindAddr+withdrawEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})

			s.Run("not enough balance", func() {
				trans := model.Transaction{
					TargetWalletID: &wallet2.ID,
					Currency:       currencyUSD,
					Sum:            3000,
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPut,
					bindAddr+withdrawEndpoint,
					trans,
					nil)

				s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
			})
		})

		s.Run("200", func() {
			trans := model.Transaction{
				TargetWalletID: &wallet2.ID,
				Currency:       currencyUSD,
				Sum:            300,
			}

			var transferResponse apiserver.TransferResponse

			resp := s.sendRequest(
				context.Background(),
				http.MethodPut,
				bindAddr+withdrawEndpoint,
				trans,
				&apiserver.HTTPResponse{Data: &transferResponse})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(transferResponse.TransactionID)
			s.transactions = append(s.transactions, transferResponse.TransactionID)
		})
	})

	s.Run("/wallets/transactions", func() {
		s.Run("200", func() {
			var transactions []model.Transaction

			resp := s.sendRequest(
				context.Background(),
				http.MethodGet,
				bindAddr+transactionsEndpoint,
				nil,
				&apiserver.HTTPResponse{Data: &transactions})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NotZero(len(transactions))
		})
	})

}

func (s *IntegrationTestSuite) checkWalletPost(wallet *model.Wallet) {
	var respWalletData model.Wallet

	resp := s.sendRequest(context.Background(), http.MethodPost, bindAddr+walletEndpoint, wallet, &apiserver.HTTPResponse{Data: &respWalletData})
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().Equal(wallet.Currency, respWalletData.Currency)
	s.Require().Equal(wallet.OwnerID, respWalletData.OwnerID)
	s.Require().Equal(wallet.Balance, respWalletData.Balance)
	s.Require().Equal(wallet.Name, respWalletData.Name)
	s.Require().NotZero(respWalletData.ID)
	wallet.ID = respWalletData.ID
	s.wallets = append(s.wallets, wallet.ID)
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
