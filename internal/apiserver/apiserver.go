package apiserver

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type APIServer struct {
	router  *chi.Mux
	cfg     Config
	server  *http.Server
	service service
	key     *rsa.PublicKey
	metrics metrics
}

type metrics interface {
	TrackHTTPRequest(start time.Time, r *http.Request)
}

type Config struct {
	BindAddress string
}

func New(cfg Config, service service, key *rsa.PublicKey, metrics metrics) *APIServer {
	router := chi.NewRouter()

	return &APIServer{
		cfg:     cfg,
		service: service,
		router:  router,
		key:     key,
		metrics: metrics,
		server: &http.Server{
			Addr:              cfg.BindAddress,
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           router,
		},
	}
}

func (s *APIServer) Run(ctx context.Context) error {
	zap.L().Debug("starting api server")
	defer zap.L().Info("server stopped")

	s.configRouter()

	zap.L().Debug("configured router")

	go func() {
		<-ctx.Done()

		zap.L().Debug("closing server")

		gfCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		zap.L().Debug("attempting graceful shutdown")

		//nolint: contextcheck
		if err := s.server.Shutdown(gfCtx); err != nil {
			zap.L().With(zap.Error(err)).Warn("failed to gracefully shutdown server")

			return
		}
	}()

	zap.L().Info("sever starting", zap.String("port", s.cfg.BindAddress))

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("s.server.ListenAndServe(): %w", err)
	}

	return nil
}

func (s *APIServer) configRouter() {
	zap.L().Debug("configuring router")

	s.router.Route("/api", func(r chi.Router) {
		r.Use(s.Metrics)
		r.Use(s.JWTAuth)

		r.Route("/v1", func(r chi.Router) {
			r.Post("/wallets", s.createWallet)
			r.Get("/wallets", s.getWallets)
			r.Get("/wallets/{id}", s.getWalletByID)
			r.Delete("/wallets/{id}", s.deleteWallet)
			r.Patch("/wallets/{id}", s.updateWallet)

			r.Put("/wallets/transfer", s.transfer)
			r.Put("/wallets/deposit", s.deposit)
			r.Put("/wallets/withdraw", s.withdraw)

			r.Get("/wallets/transactions", s.getTransactions)
		})
	})

	s.router.Get("/metrics", promhttp.Handler().ServeHTTP)
}
