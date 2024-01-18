package main

import (
	"github.com/Saaghh/wallet/internal/apiserver"
	"github.com/Saaghh/wallet/internal/config"
	"github.com/Saaghh/wallet/internal/logger"
	"github.com/sirupsen/logrus"
)

var Logger logrus.Logger

func main() {

	cfg := config.New()

	if err := logger.InitLogger(cfg.LogLevel); err != nil {
		logrus.Panic(err)
	}

	s := apiserver.New(apiserver.APIServerConfig{
		Port: cfg.Port,
	})

	if err := s.Run(); err != nil {
		logrus.Panic(err)
	}
}
