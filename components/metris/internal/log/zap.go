package log

import (
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog"
)

type zapLogger struct {
	logger *zap.SugaredLogger
}

var (
	_        Logger          = (*zapLogger)(nil)
	logLevel zap.AtomicLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
)

func NewZapLogger() Logger {
	conf := zap.Config{
		Level:         logLevel,
		Development:   false,
		DisableCaller: true,
		Encoding:      "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	opts := []zap.Option{
		zap.AddCallerSkip(1), // prevent from always showing the logger as the caller
	}

	logger, err := conf.Build(opts...)
	if err != nil {
		panic(err)
	}

	// capture global zap logging.
	_ = zap.ReplaceGlobals(logger)

	// capture standard golang "log".
	_, err = zap.RedirectStdLogAt(logger, zapcore.DebugLevel)
	if err != nil {
		panic(err)
	}

	// capture klog logs and write them at debug level because some dependencies are using it.
	if loggerwriter, logerr := zap.NewStdLogAt(logger, zapcore.DebugLevel); logerr == nil {
		klog.SetOutput(loggerwriter.Writer())
	}

	return &zapLogger{
		logger: logger.Sugar(),
	}
}

func (l *zapLogger) GetLogLevel() string {
	return logLevel.String()
}

// SetLogLevel sets the log output level.
func (l *zapLogger) SetLogLevel(level string) {
	var loglevel zapcore.Level

	err := loglevel.UnmarshalText([]byte(level))
	if err != nil {
		return
	}

	logLevel.SetLevel(loglevel)
}

func (l *zapLogger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *zapLogger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *zapLogger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *zapLogger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *zapLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *zapLogger) Panic(args ...interface{}) {
	l.logger.Panic(args...)
}

func (l *zapLogger) Panicf(format string, args ...interface{}) {
	l.logger.Panicf(format, args...)
}

func (l *zapLogger) Print(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *zapLogger) Println(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *zapLogger) Printf(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *zapLogger) With(args ...interface{}) Logger {
	return &zapLogger{
		logger: l.logger.With(args...),
	}
}

func (l *zapLogger) Named(name string) Logger {
	return &zapLogger{
		logger: l.logger.Named(name),
	}
}

func (l *zapLogger) Flush() {
	err := l.logger.Sync()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (l *zapLogger) LevelHandler(w http.ResponseWriter, r *http.Request) {
	logLevel.ServeHTTP(w, r)
}
