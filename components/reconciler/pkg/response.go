package mothership

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

func ReadErrResponse(reader io.Reader) (HTTPErrorResponse, error) {
	decoder := json.NewDecoder(reader)
	response := HTTPErrorResponse{}
	err := decoder.Decode(&response)

	return response, err
}

func (httpErr HTTPErrorResponse) ToError(statusCode int) error {
	var err error
	switch statusCode {
	case http.StatusForbidden:
		{
			err = errors.Errorf("Request can't be fulfilled, reason: %s", httpErr.Error)
		}
	case http.StatusInternalServerError:
		{
			err = errors.Errorf("Request can't be fulfilled by server, reason: %s", httpErr.Error)
		}
	case http.StatusNotFound:
		{
			err = errors.Errorf("Given object couldn't be found, reason: %s", httpErr.Error)
		}
	default:
		err = errors.Errorf("Unhandled status code: %d, reason: %s", statusCode, httpErr.Error)
	}

	return err
}
