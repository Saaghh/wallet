package apiserver

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type APIserver struct {
	config *Config
	logger *logrus.Logger
	router *chi.Mux
}

func New(config *Config) *APIserver {
	return &APIserver{
		config: config,
		logger: logrus.New(),
		router: chi.NewRouter(),
	}
}

func (s *APIserver) Start() error {
	if err := s.configLogger(); err != nil {
		return err
	}

	s.configRouter()

	s.logger.Info("api server successfully started")

	return http.ListenAndServe(s.config.Port, s.router)
}

func (s *APIserver) configLogger() error {
	level, err := logrus.ParseLevel(s.config.LogLevel)
	if err != nil {
		return err
	}

	s.logger.SetLevel(level)

	return nil
}

func (s *APIserver) configRouter() {
	s.router.Get("/time", s.handleTime)
}

func (s *APIserver) handleTime(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(time.Now().String())); err != nil {
			panic(err)
		}

		return
	}
}
