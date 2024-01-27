package main

import (
	"context"
	migrate "github.com/rubenv/sql-migrate"
	"os/signal"
	"syscall"

	"github.com/Saaghh/wallet/internal/apiserver"
	"github.com/Saaghh/wallet/internal/config"
	"github.com/Saaghh/wallet/internal/logger"
	"github.com/Saaghh/wallet/internal/service"
	"github.com/Saaghh/wallet/internal/store"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	cfg := config.New()

	logger.InitLogger(logger.Config{Level: cfg.LogLevel})

	str, err := store.New(ctx, cfg)
	if err != nil {
		zap.L().With(zap.Error(err)).Panic("str.New")
	}

	if err := str.Migrate(migrate.Up); err != nil {
		zap.L().With(zap.Error(err)).Panic("str.Migrate")
	}

	zap.L().Info("successful migration")

	srv := service.New(str)

	defer zap.L().Sync()
	// TODO add error checking. Currently always errors with no obvious reason

	s := apiserver.New(apiserver.Config{
		BindAddress: cfg.BindAddress,
	}, srv)

	if err := s.Run(ctx); err != nil {
		zap.L().Panic(err.Error())
	}
}
