package main

import (
	"github.com/Saaghh/wallet/internal/apiserver"
	"github.com/Saaghh/wallet/internal/config"
	"github.com/Saaghh/wallet/internal/logger"
	"go.uber.org/zap"
)

func main() {
	cfg := config.New()

	logger.InitLogger()
	defer zap.L().Sync()

	s := apiserver.New(apiserver.Config{
		Port: cfg.Port,
	})

	if err := s.Run(); err != nil {
		zap.L().Panic(err.Error())
	}
}
