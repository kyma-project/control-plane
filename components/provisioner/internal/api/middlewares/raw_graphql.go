package middlewares

import (
	"bytes"
	"fmt"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"io"
	"net/http"
	"strings"
)

const graphqlPath = "/graphql"
const provisionOperationName = "provisionRuntime"

func ExtractRawGraphQL(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, graphqlPath) {
			bodyData, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(bodyData))
			stringBodyData := string(bodyData)

			if isOperationProvisioning(stringBodyData) {
				// save gql call to PV
				shootName := getShootName(stringBodyData)
				err := util.WriteToPV(stringBodyData, shootName)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
		}

		handler.ServeHTTP(w, r)
	})
}

func getShootName(gqlData string) string {
	start := strings.Index(gqlData, `name: \"`)
	if start == -1 {
		fmt.Println("No match found")
		return ""
	}
	start += len(`"name":"`)

	end := strings.Index(gqlData[start:], `\"`)
	if end == -1 {
		fmt.Println("No match found")
		return ""
	}

	return gqlData[start : start+end]
}

func isOperationProvisioning(gqlData string) bool {
	mutationStart := strings.Index(gqlData, provisionOperationName)
	if mutationStart == -1 {
		fmt.Println("No match found")
		return false
	}

	mutationStart += len(provisionOperationName)

	openParen := strings.Index(gqlData[mutationStart:], "(")
	if openParen == -1 {
		fmt.Println("No match found")
		return false
	}

	return true
}
