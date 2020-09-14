package log

import "net/http"

type noopLogger struct {
	loglevel string
}

var _ Logger = (*noopLogger)(nil)

func NewNoopLogger() Logger {
	return &noopLogger{}
}

func (l *noopLogger) GetLogLevel() string                       { return l.loglevel }
func (l *noopLogger) SetLogLevel(level string)                  { l.loglevel = level }
func (l *noopLogger) Debug(args ...interface{})                 {}
func (l *noopLogger) Debugf(format string, args ...interface{}) {}
func (l *noopLogger) Info(args ...interface{})                  {}
func (l *noopLogger) Infof(format string, args ...interface{})  {}
func (l *noopLogger) Warn(args ...interface{})                  {}
func (l *noopLogger) Warnf(format string, args ...interface{})  {}
func (l *noopLogger) Error(args ...interface{})                 {}
func (l *noopLogger) Errorf(format string, args ...interface{}) {}
func (l *noopLogger) Fatal(args ...interface{})                 {}
func (l *noopLogger) Fatalf(format string, args ...interface{}) {}
func (l *noopLogger) Panic(args ...interface{})                 {}
func (l *noopLogger) Panicf(format string, args ...interface{}) {}
func (l *noopLogger) Println(args ...interface{})               {}
func (l *noopLogger) With(args ...interface{}) Logger           { return l }
func (l *noopLogger) Named(name string) Logger                  { return l }
func (l *noopLogger) Flush()                                    {}
func (l *noopLogger) LevelHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
