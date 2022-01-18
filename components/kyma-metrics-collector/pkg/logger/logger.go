package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// OutputFormatJSON is used to define output format as json
	OutputFormatJSON OutputFormat = "json"

	// OutputFormatPlain is used to define output format as plain text
	OutputFormatPlain OutputFormat = "plain"

	// KeyError is used as a named key for a log message with error.
	KeyError = "error"

	// KeyResult is used as a named key for a log message with result.
	KeyResult = "result"

	// KeyReason is used as a named key for a log message with reason.
	KeyReason = "reason"

	// KeyStep is used as a named key for a log message with step.
	KeyStep = "step"

	// keyAction is used as a named key for a log message with action.
	keyAction = "action"

	// keyVersion is used as a named key for a log message with version.
	keyVersion = "version"

	// ValueFail is used as a value for a log message with failure.
	ValueFail = "fail"

	// ValueSuccess is used as a value for a log message with success.
	ValueSuccess = "success"
)

var outputFormat = OutputFormatJSON

type OutputFormat string

func NewLogger(debug bool) *zap.SugaredLogger {
	logLevel := zapcore.InfoLevel
	if debug {
		logLevel = zapcore.DebugLevel
	}
	return newLogger(logLevel).Sugar()
}

func newLogger(logLevel zapcore.Level) *zap.Logger {
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:   "message",
		LevelKey:     "level",
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		TimeKey:      "time",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		CallerKey:    "caller",
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	switch outputFormat {
	case OutputFormatPlain:
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	case OutputFormatJSON:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	return zap.New(
		zapcore.NewCore(
			encoder,
			zapcore.Lock(os.Stderr),
			zap.NewAtomicLevelAt(logLevel),
		),
		zap.ErrorOutput(os.Stderr))
}

// pair represents a log key/value pair.
type pair [2]string

// LoggerOpt represents a function that returns a log pair instance when executed.
type LoggerOpt func() pair

// WithAction returns a LoggerOpt for the given action.
func WithAction(action string) LoggerOpt {
	return func() pair {
		return pair{keyAction, action}
	}
}

// SetOutputFormat sets the log output format
func SetOutputFormat(of OutputFormat) {
	outputFormat = of
}
