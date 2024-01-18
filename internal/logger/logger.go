package logger

import (
	"go.uber.org/zap"
)

func InitLogger() {
	logger := zap.Must(zap.NewDevelopment())

	zap.ReplaceGlobals(logger)
}
