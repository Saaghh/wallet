package apiserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Saaghh/wallet/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type APIServer struct {
	router  *chi.Mux
	cfg     Config
	server  *http.Server
	service *service.Service
}

type Config struct {
	BindAddress string
}

func New(cfg Config, service *service.Service) *APIServer {
	router := chi.NewRouter()
	return &APIServer{
		cfg:     cfg,
		service: service,
		router:  router,
		server: &http.Server{
			Addr:              cfg.BindAddress,
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           router,
		},
	}
}

func (s *APIServer) Run(ctx context.Context) error {
	zap.L().Info("starting api server")
	defer zap.L().Info("server stopped")

	s.configRouter()

	zap.L().Debug("configured router")

	go func() {
		<-ctx.Done()

		zap.L().Debug("closing server")

		gfCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		zap.L().Debug("attempting graceful shutdown")

		if err := s.server.Shutdown(gfCtx); err != nil {
			zap.L().With(zap.Error(err)).Warn("failed to gracefully shutdown http server")

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
	s.router.Get("/time", s.handleTime)
	s.router.Get("/visitHistory", s.handleVisitHistory)
}
