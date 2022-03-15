package mothership

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

func TestReadErrResponse(t *testing.T) {
	t.Run("Valid JSON", func(t *testing.T) {
		//GIVEN
		payload := `{"error":"error message"}`

		//WHEN
		errResposne, err := ReadErrResponse(strings.NewReader(payload))

		//THEN
		require.NoError(t, err)
		require.Equal(t, errResposne.Error, "error message")
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		//GIVEN
		payload := `{"error":"error message}`

		//WHEN
		_, err := ReadErrResponse(strings.NewReader(payload))

		//THEN
		require.Error(t, err)
	})
}

func TestToError(t *testing.T) {
	testCases := map[string]struct {
		input       HTTPErrorResponse
		statusCode  int
		expectedErr string
	}{
		"Status Forbidden": {
			statusCode:  http.StatusForbidden,
			input:       HTTPErrorResponse{Error: "test error"},
			expectedErr: "Request can't be fulfilled, reason: test error",
		},
		"Status Internal Server Error": {
			statusCode:  http.StatusInternalServerError,
			input:       HTTPErrorResponse{Error: "test internal error"},
			expectedErr: "Request can't be fulfilled by server, reason: test internal error",
		},
		"Status Not Found Error": {
			statusCode:  http.StatusNotFound,
			input:       HTTPErrorResponse{Error: "test not found error"},
			expectedErr: "Given object couldn't be found, reason: test not found error",
		},
		"Status unknown": {
			statusCode:  http.StatusConflict,
			input:       HTTPErrorResponse{Error: "test unknown error"},
			expectedErr: "Unhandled status code: 409, reason: test unknown error",
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			//GIVEN

			//WHEN
			err := testCase.input.ToError(testCase.statusCode)
			//THEN
			require.Error(t, err)
			require.Equal(t, err.Error(), testCase.expectedErr)
		})
	}

}
