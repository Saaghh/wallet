package apiserver

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type APIServer struct {
	router *chi.Mux
	cfg    Config
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

func (s *APIServer) Run() error {
	s.configRouter()

	zap.L().Info("api server successfully started")

	return http.ListenAndServe(s.cfg.Port, s.router)
}

func (s *APIServer) configRouter() {
	s.router.Get("/time", s.handleTime)
}

func (s *APIServer) handleTime(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(time.Now().String())); err != nil {
			zap.L().Panic(err.Error())
		}

		return
	}
}
