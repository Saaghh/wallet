package apiserver

import "github.com/sirupsen/logrus"

type APIserver struct {
	config *Config
	logger *logrus.Logger
}

func New(config *Config) *APIserver {
	return &APIserver{
		config: config,
		logger: logrus.New(),
	}
}

func (s *APIserver) Start() error {

	if err := s.configLogger(); err != nil {
		return err
	}

	s.logger.Info("api server successfully started")

	return nil
}

func (s *APIserver) configLogger() error {
	level, err := logrus.ParseLevel(s.config.LogLevel)
	if err != nil {
		return nil
	}

	s.logger.SetLevel(level)

	return nil
}
