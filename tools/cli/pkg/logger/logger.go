package logger

import (
	"github.com/int128/kubelogin/pkg/infrastructure/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

// CfgLevel is the configured logging level
var CfgLevel int

// New returns a Logger with the standard log.Logger
func New() Logger {
	log := logrus.New()
	log.Level = logrus.Level(CfgLevel)
	return &logging{
		FieldLogger: log,
		verbosity:   CfgLevel,
	}
}

// Logger is the interface to interact with a CLI logging instance
type Logger interface {
	logrus.FieldLogger
	AddFlags(f *pflag.FlagSet)
	V(level int) logger.Verbose
	IsEnabled(level int) bool
}

// logging provides logging facility using log.Logger and klog.
type logging struct {
	logrus.FieldLogger
	verbosity int
}

type verbose struct {
	l     *logging
	level int
}

// AddFlags adds the flags such as -v.
func (l *logging) AddFlags(f *pflag.FlagSet) {
	f.IntVarP(&l.verbosity, "verbose", "v", 0, "Option that turns verbose logging to stderr. Valid values are 0 (default) - 6 (maximum verbosity).")
}

func AddFlags(f *pflag.FlagSet) {
	f.IntVarP(&CfgLevel, "verbose", "v", 0, "Option that turns verbose logging to stderr. Valid values are 0 (default) - 6 (maximum verbosity).")
}

// V returns a logger enabled only if the level is enabled.
func (l *logging) V(level int) logger.Verbose {
	v := &verbose{l: l, level: level}
	return v
}

// IsEnabled returns true if the level is enabled.
func (l *logging) IsEnabled(level int) bool {
	return l.verbosity >= level
}

// Infof logs a verbose info message with he given format and arguments based on the configured verbosity
func (v *verbose) Infof(format string, args ...interface{}) {
	if v.l.verbosity >= v.level {
		v.l.Infof(format, args...)
	}
}
