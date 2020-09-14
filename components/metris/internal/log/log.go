package log

import "net/http"

var DefaultLogger Logger = NewZapLogger()

type Logger interface {
	GetLogLevel() string
	SetLogLevel(level string)
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Println(args ...interface{}) // for prometheus
	Flush()
	Named(name string) Logger
	With(args ...interface{}) Logger
	LevelHandler(w http.ResponseWriter, r *http.Request)
}

type Level string

const (
	// DebugLevel logs a lot, and shot be disabled in production.
	DebugLevel = "debug"
	// InfoLevel is the default logging priority.
	InfoLevel = "info"
	// WarnLevel logs are more important than Info, but don't need user attention right away.
	WarnLevel = "warn"
	// ErrorLevel logs are high-priority.
	ErrorLevel = "error"
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel = "fatal"
	// PanicLevel logs a message, then panics.
	PanicLevel = "panic"
)

func GetLogLevel() string {
	return DefaultLogger.GetLogLevel()
}

// SetLogLevel sets the log output level.
func SetLogLevel(level string) {
	DefaultLogger.SetLogLevel(level)
}

// Debug construct and log a message.
func Debug(args ...interface{}) {
	DefaultLogger.Debug(args...)
}

// Debugf formats the log message according to a format.
func Debugf(format string, args ...interface{}) {
	DefaultLogger.Debugf(format, args...)
}

// Info construct and log a message.
func Info(args ...interface{}) {
	DefaultLogger.Info(args...)
}

// Infof formats the log message according to a format.
func Infof(format string, args ...interface{}) {
	DefaultLogger.Infof(format, args...)
}

// Warn construct and log a message.
func Warn(args ...interface{}) {
	DefaultLogger.Warn(args...)
}

// Warnf formats the log message according to a format.
func Warnf(format string, args ...interface{}) {
	DefaultLogger.Warnf(format, args...)
}

// Error construct and log a message.
func Error(args ...interface{}) {
	DefaultLogger.Error(args...)
}

// Errorf formats the log message according to a format.
func Errorf(format string, args ...interface{}) {
	DefaultLogger.Errorf(format, args...)
}

// Fatal construct and log a message, then calls os.Exit.
func Fatal(args ...interface{}) {
	DefaultLogger.Fatal(args...)
}

// Fatalf formats the log message according to a format, then calls os.Exit.
func Fatalf(format string, args ...interface{}) {
	DefaultLogger.Fatalf(format, args...)
}

// Panic construct and log a message, then panics.
func Panic(args ...interface{}) {
	DefaultLogger.Panic(args...)
}

// Panicf formats the log message according to a format, then panics.
func Panicf(format string, args ...interface{}) {
	DefaultLogger.Fatalf(format, args...)
}

func Println(args ...interface{}) {
	DefaultLogger.Println(args)
}

// Flush flushes any buffered log entries.
func Flush() {
	DefaultLogger.Flush()
}

// Named adds a sub-scope to the logger's name.
func Named(name string) Logger {
	return DefaultLogger.Named(name)
}

// With adds values to the logging context.
func With(args ...interface{}) Logger {
	return DefaultLogger.With(args...)
}

// LevelHandler is a simple JSON endpoint that can report on or change the current logging level.
//
// GET requests return a JSON description of the current logging level.
// PUT requests change the logging level and expect a payload like: {"level":"info"}
func LevelHandler(w http.ResponseWriter, r *http.Request) {
	DefaultLogger.LevelHandler(w, r)
}
