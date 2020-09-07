package testkit

import (
	"strings"
	"testing"
)

type Logger struct {
	t            *testing.T
	fields       []string
	joinedFields string
}

func NewLogger(t *testing.T, fields ...string) *Logger {
	joinedFields := strings.Join(fields, " ")

	return &Logger{
		t:            t,
		fields:       fields,
		joinedFields: joinedFields,
	}
}

func (l Logger) Log(msg string) {
	l.t.Logf("%s   %s", msg, l.joinedFields)
}

func (l Logger) Logf(format string, msg ...interface{}) {
	format = strings.Join([]string{format, "%s"}, "\t")
	msg = append(msg, l.joinedFields)
	l.t.Logf(format, msg...)
}

func (l Logger) Error(msg string) {
	l.t.Errorf("%s\t%s", msg, l.joinedFields)
}

func (l Logger) Errorf(format string, msg ...interface{}) {
	format = strings.Join([]string{format, "%s"}, "\t")
	msg = append(msg, l.joinedFields)
	l.t.Errorf(format, msg...)
}

func (l *Logger) AddField(field string) {
	l.fields = append(l.fields, field)
	l.joinedFields = strings.Join(l.fields, " ")
}
