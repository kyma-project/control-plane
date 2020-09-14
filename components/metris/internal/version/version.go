package version

import (
	"bytes"
	"html/template"
	"runtime"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/metris/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Build information. Populated at build-time.

var (
	Version    string = "dev"
	CommitHash string
	BuildDate  string = time.Now().UTC().String()
	GoVersion         = runtime.Version()

	MetrisBuildInfo = promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "info",
			Help:      "A metric with a constant '1' value labeled by version, commit, and goversion from which metris was built.",
			ConstLabels: prometheus.Labels{
				"version":   Version,
				"commit":    CommitHash,
				"goversion": GoVersion,
			},
		},
		func() float64 { return 1 },
	)
)

// versionInfoTmpl contains the template used by Info.
var versionInfoTmpl = `
{{.program}}
  version:      {{.version}}
  commit:       {{.commit}}
  build date:   {{.buildDate}}
  go version:   {{.goVersion}}
`

// Print returns version information.
func Print() string {
	m := map[string]string{
		"program":   "metris",
		"version":   Version,
		"commit":    CommitHash,
		"buildDate": BuildDate,
		"goVersion": GoVersion,
	}

	t := template.Must(template.New("version").Parse(versionInfoTmpl))

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "version", m); err != nil {
		panic(err)
	}

	return strings.TrimSpace(buf.String())
}
