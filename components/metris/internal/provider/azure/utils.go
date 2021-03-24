package azure

import (
	"context"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
)

type ctxKey int

const timeKey = ctxKey(1)

// LogRequest inspect the request being made to Azure API and writes it to the logger.
func LogRequest(logger log.Logger) autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			if logger.GetLogLevel() == log.DebugLevel {
				r = r.WithContext(context.WithValue(r.Context(), timeKey, time.Now()))
				if d, e := httputil.DumpRequestOut(r, false); e == nil {
					logger.Debug(string(d))
				}
			}
			return p.Prepare(r)
		})
	}
}

// LogResponse inspect the response received from Azure API, process metrics and writes it to the logger.
func LogResponse(logger log.Logger) autorest.RespondDecorator {
	return func(r autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(resp *http.Response) error {
			if resp != nil {
				provider, resource := parseServiceURL(resp.Request.URL.Path)
				apiRequestCounter.WithLabelValues(provider, resource, strconv.Itoa(resp.StatusCode)).Inc()

				if logger.GetLogLevel() == log.DebugLevel {
					if start, ok := resp.Request.Context().Value(timeKey).(time.Time); ok {
						logger.
							With("path", resp.Request.URL.Path).
							With("status", resp.StatusCode).
							With("time", time.Since(start)).
							Debug("request")
					}

					if dump, e := httputil.DumpResponse(resp, false); e == nil {
						logger.Debug(string(dump))
					}
				}
			}

			return r.Respond(resp)
		})
	}
}

// parseServiceURL parses a Azure REST URL path to retrieve the provider and service called.
func parseServiceURL(path string) (provider, resource string) {
	var (
		servicePattern *regexp.Regexp = regexp.MustCompile(`(?i)subscriptions/.+/providers/(.+?)/(.+?)$`)
		match          []string       = servicePattern.FindStringSubmatch(path)
	)

	provider = "unknown"
	resource = "unknown"

	if len(match) > 0 {
		provider = strings.ReplaceAll(strings.ToLower(match[1]), "microsoft.", "")
		resource = strings.ToLower(match[2])
	} else if _, err := regexp.Match(`(?i)subscriptions/.+/resourcegroups/?[^/]*$`, []byte(path)); err == nil {
		provider = "resources"
		resource = "groups"
	}

	return provider, resource
}
