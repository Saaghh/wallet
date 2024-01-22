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

	server := &http.Server{
		Addr:              cfg.BindAddress,
		Handler:           chi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &APIServer{
		cfg:     cfg,
		service: service,
		server:  server,
	}
}

func (s *APIServer) Run(ctx context.Context) error {
	zap.L().Info("starting api server")

	s.configRouter()

	go func() {
		<-ctx.Done()

		gfCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.server.Shutdown(gfCtx); err != nil {
			zap.L().With(zap.Error(err)).Warn("failed to gracefully shutdown http server")

			return
		}

		zap.L().Info("server successfully stopped")
	}()

	zap.L().Info("sever starting", zap.String("port", s.cfg.BindAddress))

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("s.server.ListenAndServe(): %w", err)
	}

	return nil
}

func (s *APIServer) configRouter() {
	s.router.Get("/time", s.handleTime)
	s.router.Get("/visitHistory", s.handleVisitHistory)
}
