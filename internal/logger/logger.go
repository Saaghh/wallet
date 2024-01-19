package logger

import (
	log "go.uber.org/zap"
)

func InitLogger() {
	logger := log.Must(log.NewDevelopment())

	log.ReplaceGlobals(logger)

	log.L().Info("successful logger initialization")
}
