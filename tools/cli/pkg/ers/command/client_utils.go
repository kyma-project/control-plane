package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/tools/cli/pkg/command"
)

func IsErrResponse(statusCode int) bool {
	return statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices
}

func ResponseErr(resp *http.Response) error {
	msg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg = []byte(fmt.Errorf("unexpected error: %w", err).Error())
	}
	return fmt.Errorf("%s %d: %w", command.ErrMothershipResponse, string(msg), resp.StatusCode)
}

func OpString(op *runtime.Operation) string {
	if op == nil {
		return "No Operation"
	}
	return fmt.Sprintf("%+v", *op)
}

func ToJson(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return "", fmt.Errorf("while creating json: %w", err)
	}

	return string(data), nil
}
