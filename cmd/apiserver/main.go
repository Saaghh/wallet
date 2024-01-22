package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/Saaghh/wallet/internal/apiserver"
	"github.com/Saaghh/wallet/internal/config"
	"github.com/Saaghh/wallet/internal/logger"
	"github.com/Saaghh/wallet/internal/service"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	cfg := config.New()
	service := service.New()

	logger.InitLogger(logger.Config{Level: cfg.LogLevel})

	defer zap.L().Sync()
	// TODO add error checking. Currently always errors with no obvious reason

	s := apiserver.New(apiserver.Config{
		BindAddress: cfg.BindAddress,
	}, service)

	if err := s.Run(ctx); err != nil {
		zap.L().Panic(err.Error())
	}
}
