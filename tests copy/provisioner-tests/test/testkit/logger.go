package testkit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	t *testing.T
	l *logrus.Entry
}

func NewLogger(t *testing.T, fields logrus.Fields) *Logger {
	return &Logger{
		t: t,
		l: logrus.WithFields(fields),
	}
}

func (l Logger) Log(msg string) {
	l.l.Info(msg)
}

func (l Logger) Logf(format string, msg ...interface{}) {
	l.l.Infof(format, msg...)
}

func (l Logger) Error(msg string) {
	l.t.Errorf("%s %s", l.joinedFields(), msg)
}

func (l Logger) Errorf(format string, msg ...interface{}) {
	msg = append(msg, l.joinedFields())
	l.t.Errorf("%s %s", l.joinedFields(), fmt.Sprintf(format, msg...))
}

func (l *Logger) WithField(key string, value interface{}) {
	l.l.WithField(key, value)
}

func (l Logger) joinedFields() string {
	fields := []string{}
	for key, value := range l.l.Data {
		fields = append(fields, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(fields, " ")
}
