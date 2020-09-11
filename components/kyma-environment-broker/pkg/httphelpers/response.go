package httphelpers

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

func WriteResponse(w http.ResponseWriter, code int, object interface{}) {
	data, err := json.Marshal(object)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(data)
	if err != nil {
		logrus.Warnf("could not write response %s", string(data))
	}
}

type errObj struct {
	Error string `json:"error"`
}

func WriteErrorResponse(w http.ResponseWriter, code int, err error) {
	WriteResponse(w, code, errObj{Error: err.Error()})
}
