package apiserver

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type APIServer struct {
	router *chi.Mux
	cfg    APIServerConfig
}

type APIServerConfig struct {
	Port string
}

func New(cfg APIServerConfig) *APIServer {
	return &APIServer{
		cfg:    cfg,
		router: chi.NewRouter(),
	}
}

func (s *APIServer) Run() error {
	s.configRouter()

	logrus.Info("api server successfully started")

	return http.ListenAndServe(s.cfg.Port, s.router)
}

func (s *APIServer) configRouter() {
	s.router.Get("/time", s.handleTime)
}

func (s *APIServer) handleTime(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(time.Now().String())); err != nil {
			logrus.Panic(err)
		}

		return
	}
}
