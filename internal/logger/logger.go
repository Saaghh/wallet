package logger

import "github.com/sirupsen/logrus"

func InitLogger(logLevel string) error {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}

	logrus.SetLevel(level)

	return nil
}
