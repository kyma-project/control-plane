package middleware

import (
	"fmt"
	"net/http"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/sirupsen/logrus"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				logrus.Errorf("Panic when %s was called: %+v", r.URL, err)
				httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("Internal error, unable to handle the request"))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
