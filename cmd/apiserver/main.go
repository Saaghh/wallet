package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/Saaghh/wallet/internal/apiserver"
	"github.com/Saaghh/wallet/internal/config"
	"github.com/Saaghh/wallet/internal/logger"
	"github.com/Saaghh/wallet/internal/service"
	"github.com/Saaghh/wallet/internal/service/currconv"
	"github.com/Saaghh/wallet/internal/store"
	migrate "github.com/rubenv/sql-migrate"
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

	converter := currconv.New()

	srv := service.New(str, converter)

	// no error handling for now
	// check https://github.com/uber-go/zap/issues/991
	//nolint: errcheck
	defer zap.L().Sync()

	s := apiserver.New(apiserver.Config{
		BindAddress: cfg.BindAddress,
	}, srv)

	if err := s.Run(ctx); err != nil {
		zap.L().Panic(err.Error())
	}
}
