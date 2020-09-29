package logger

import (
	"flag"
	"log"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/klog"
)

// New returns a Logger with the standard log.Logger and klog.
func New() Logger {
	return &logger{
		goLogger: log.New(os.Stderr, "", 0),
	}
}

type Logger interface {
	AddFlags(f *pflag.FlagSet)
	Printf(format string, args ...interface{})
	V(level int) klog.Verbose
	IsEnabled(level int) bool
}

type goLogger interface {
	Printf(format string, v ...interface{})
}

// Logger provides logging facility using log.Logger and klog.
type logger struct {
	goLogger
}

// AddFlags adds the flags such as -v.
func (*logger) AddFlags(f *pflag.FlagSet) {
	gf := flag.NewFlagSet("", flag.ContinueOnError)
	klog.InitFlags(gf)
	f.AddGoFlagSet(gf)
}

// V returns a logger enabled only if the level is enabled.
func (*logger) V(level int) klog.Verbose {
	return klog.V(klog.Level(level))
}

// IsEnabled returns true if the level is enabled.
func (*logger) IsEnabled(level int) bool {
	return bool(klog.V(klog.Level(level)))
}
