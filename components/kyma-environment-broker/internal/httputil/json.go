package httputil

import (
	"encoding/json"
	"net/http"
)

// JSONEncode encodes the given object to json format and writes it to given ResponseWriter
func JSONEncode(rw http.ResponseWriter, v interface{}) error {
	rw.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(rw).Encode(v)
}

// JSONEncodeWithCode encodes the given object to json format and writes it to given ResponseWriter
// with custom status code
func JSONEncodeWithCode(rw http.ResponseWriter, v interface{}, statusCode int) error {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	return json.NewEncoder(rw).Encode(v)
}
