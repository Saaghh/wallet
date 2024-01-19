package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/Saaghh/wallet/internal/apiserver"
	"github.com/Saaghh/wallet/internal/config"
	"github.com/Saaghh/wallet/internal/logger"
	log "go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	cfg := config.New()

	logger.InitLogger()

	defer func() {
		if err := log.L().Sync(); err != nil {
			panic(err)
		}
	}()

	s := apiserver.New(apiserver.Config{
		Port: cfg.Port,
	})

	if err := s.Run(ctx); err != nil {
		log.L().Panic(err.Error())
	}
}
