package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	"github.com/kyma-project/control-plane/tools/cli/pkg/command"
	"github.com/pkg/errors"
)

func IsErrResponse(statusCode int) bool {
	return statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices
}

func ResponseErr(resp *http.Response) error {
	msg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg = []byte(errors.Wrap(err, "unexpected error").Error())
	}
	return errors.Wrapf(command.ErrMothershipResponse, "%s %d", string(msg), resp.StatusCode)
}

func StateCreatedFormatted(obj interface{}) string {
	state := obj.(mothership.HTTPClusterStateResponse)
	return state.Status.Created.Format("2006/01/02 15:04:05")
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
		return "", errors.Wrapf(err, "while creating json")
	}

	return string(data), nil
}
