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

	// KeySubAccountID is used as a named key for a log message with subaccount ID.
	KeySubAccountID = "SubAccountID"

	// KeyRuntimeID is used as a named key for a log message with runtime ID.
	KeyRuntimeID = "RuntimeID"

	// KeyWorkerID is used as a named key for a log message with worker ID.
	KeyWorkerID = "WorkerID"

	// KeyRetry is used as named key for a log message which indicates the step will be retried
	KeyRetry = "willRetry"

	// KeyRequeue is used as named key for a log message which indicates that it will be requeued
	KeyRequeue = "requeue"

	// ValueFail is used as a value for a log message with failure.
	ValueFail = "fail"

	// ValueSuccess is used as a value for a log message with success.
	ValueSuccess = "success"

	// ValueTrue is used as a value for a message with true status
	ValueTrue = "true"

	// ValueFalse is used as a value for a message with true status
	ValueFalse = "false"
)

var outputFormat = OutputFormatJSON

type OutputFormat string

func NewLogger(logLevel zapcore.Level) *zap.SugaredLogger {
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

	encoderConfig = zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.MessageKey = "message"
	encoderConfig.CallerKey = "caller"

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
		zap.AddCaller(),
		zap.ErrorOutput(os.Stderr))
}

// SetOutputFormat sets the log output format
func SetOutputFormat(of OutputFormat) {
	outputFormat = of
}
