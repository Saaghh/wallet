package apiserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	log "go.uber.org/zap"
)

type APIServer struct {
	router *chi.Mux
	cfg    Config
	server *http.Server
}

type Config struct {
	Port string
}

func New(cfg Config) *APIServer {
	return &APIServer{
		cfg:    cfg,
		router: chi.NewRouter(),
	}
}

func (s *APIServer) Run(ctx context.Context) error {
	log.L().Info("starting api server")

	s.configRouter()

	go func() {
		<-ctx.Done()
		s.server.Close()
		log.L().Info("server successfully stoped")
	}()

	s.server = &http.Server{
		Addr:              s.cfg.Port,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.L().Info("sever starting", log.String("port", s.cfg.Port))

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.L().Panic(err.Error())
	}

	return nil
}

func (s *APIServer) configRouter() {
	s.router.Get("/time", s.handleTime)
}

func (s *APIServer) handleTime(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(time.Now().String())); err != nil {
			log.L().Panic(err.Error())
		}

		log.L().Info("sent /time", log.String("client", r.RemoteAddr))

		return
	}
}
