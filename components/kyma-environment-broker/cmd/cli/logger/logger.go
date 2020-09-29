package logger

import (
	"flag"
	"log"
	"os"

	"github.com/int128/kubelogin/pkg/adaptors/logger"
	"github.com/spf13/pflag"
	"k8s.io/klog"
)

// New returns a Logger with the standard log.Logger and klog.
func New() Logger {
	return &logging{
		goLogger: log.New(os.Stderr, "", 0),
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
}

// AddFlags adds the flags such as -v.
func (*logging) AddFlags(f *pflag.FlagSet) {
	gf := flag.NewFlagSet("", flag.ContinueOnError)
	klog.InitFlags(gf)
	f.AddGoFlagSet(gf)
}

// V returns a logger enabled only if the level is enabled.
func (*logging) V(level int) logger.Verbose {
	return klog.V(klog.Level(level))
}

// IsEnabled returns true if the level is enabled.
func (*logging) IsEnabled(level int) bool {
	return bool(klog.V(klog.Level(level)))
}
