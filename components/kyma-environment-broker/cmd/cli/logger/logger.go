package logger

import (
	"log"
	"os"

	"github.com/int128/kubelogin/pkg/adaptors/logger"
	"github.com/spf13/pflag"
)

// New returns a Logger with the standard log.Logger
func New() Logger {
	return &logging{
		goLogger: log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile),
	}
}

// Logger is the interface to interact with a CLI logging instance
type Logger interface {
	AddFlags(f *pflag.FlagSet)
	Printf(format string, args ...interface{})
	V(level int) logger.Verbose
	IsEnabled(level int) bool
}

type goLogger interface {
	Printf(format string, v ...interface{})
}

// logging provides logging facility using log.Logger and klog.
type logging struct {
	goLogger
	verbosity int
}

type verbose struct {
	l     *logging
	level int
}

// AddFlags adds the flags such as -v.
func (l *logging) AddFlags(f *pflag.FlagSet) {
	f.IntVarP(&l.verbosity, "verbose", "v", 0, "Option that turns verbose logging to stderr. Valid values are 0 (default) - 3 (maximum verbosity).")
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
		v.l.Printf(format, args...)
	}
}
