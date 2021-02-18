package tracing

import (
	"fmt"
	"log"

	zipkinexp "contrib.go.opencensus.io/exporter/zipkin"
	zipkin "github.com/openzipkin/zipkin-go"
	zipkinrep "github.com/openzipkin/zipkin-go/reporter"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"go.opencensus.io/trace"
)

// Config holds zipkin tracing configuration.
type Config struct {
	Enable    bool   `kong:"help='Enable tracing',name='tracing',default=false,env='TRACING_ENABLE'"`
	ZipkinURL string `kong:"help='Zipkin Collector URL',default='http://localhost:9411/api/v2/spans',env='ZIPKIN_URL'"`
}

type Tracer interface {
	Stop()
}

var tracer Tracer

func IsEnabled() bool {
	return tracer != nil
}

type zipkinTracer struct {
	reporter zipkinrep.Reporter
}

func New(conf Config) (Tracer, error) {
	t := zipkinTracer{}

	// set up a span reporter
	t.reporter = zipkinhttp.NewReporter(conf.ZipkinURL)

	// create our local service endpoint
	endpoint, err := zipkin.NewEndpoint("metris", "")
	if err != nil {
		return t, fmt.Errorf("unable to create zipkin endpoint: %w", err)
	}

	exporter := zipkinexp.NewExporter(t.reporter, endpoint)

	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	tracer = t

	return t, nil
}

func (t zipkinTracer) Stop() {
	if err := t.reporter.Close(); err != nil {
		log.Panic(err)
	}
}
