package azure

import (
	"context"
	"errors"
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
func LogRequest(logger log.Logger, tracelevel int) autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r = r.WithContext(context.WithValue(r.Context(), timeKey, time.Now()))
			r, err := p.Prepare(r)

			ctx := r.Context()
			if ctx.Err() != nil {
				var status string
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					status = strconv.Itoa(http.StatusRequestTimeout)
				} else if errors.Is(ctx.Err(), context.Canceled) {
					// 499 Client Closed Request, see https://www.nginx.com/resources/wiki/extending/api/http/
					status = "499"
				}

				provider, resource := parseServiceURL(r.URL.Path)
				apiRequestCounter.WithLabelValues(provider, resource, status).Inc()
			}

			if tracelevel > 1 && logger.GetLogLevel() == log.DebugLevel {
				if d, e := httputil.DumpRequestOut(r, tracelevel > 2); e == nil {
					logger.Debug(string(d))
				}
			}

			return r, err
		})
	}
}

// LogResponse inspect the response received from Azure API, process metrics and writes it to the logger.
func LogResponse(logger log.Logger, tracelevel int) autorest.RespondDecorator {
	return func(r autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(resp *http.Response) error {
			err := r.Respond(resp)
			if resp != nil {
				provider, resource := parseServiceURL(resp.Request.URL.Path)
				apiRequestCounter.WithLabelValues(provider, resource, strconv.Itoa(resp.StatusCode)).Inc()

				if tracelevel > 0 && logger.GetLogLevel() == log.DebugLevel {
					if start, ok := resp.Request.Context().Value(timeKey).(time.Time); ok {
						logger.
							With("path", resp.Request.URL.Path).
							With("status", resp.StatusCode).
							With("time", time.Since(start)).
							Debug("request")
					}

					if tracelevel > 1 {
						if dump, e := httputil.DumpResponse(resp, tracelevel > 2); e == nil {
							logger.Debug(string(dump))
						}
					}
				}
			}

			return err
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
