package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Server struct {
	currencies map[string]float64
	router     *chi.Mux
	server     *http.Server
	mutex      *sync.RWMutex
}

func New(bindAddr string) *Server {
	currencies := map[string]float64{
		"RUB": 1,
		"USD": 90.53,
		"EUR": 97.53,
		"KZT": 20.0115,
		"IDR": 0.00579328,
	}

	router := chi.NewRouter()

	return &Server{
		currencies: currencies,
		router:     router,
		mutex:      new(sync.RWMutex),
		server: &http.Server{
			Addr:              bindAddr,
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           router,
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	s.configRouter()

	go func() {
		<-ctx.Done()

		gfCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		//nolint: contextcheck
		if err := s.server.Shutdown(gfCtx); err != nil {
			zap.L().With(zap.Error(err)).Warn("failed to gracefully shutdown server")

			return
		}
	}()

	zap.L().Info("xr sever starting", zap.String("port", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf(" s.server.ListenAndServe(): %w", err)
	}

	return nil
}

func (s *Server) configRouter() {
	s.router.Get("/xr", s.handleGetExchangeRate)
}
