package logger

import (
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
}

// For returns a logger.
func For(pkg, fn string) *logrus.Entry {
	return logrus.
		WithField("pkg", pkg).
		WithField("fn", fn)
}
