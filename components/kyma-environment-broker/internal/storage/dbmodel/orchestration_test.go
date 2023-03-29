package dbmodel

import (
	"encoding/json"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/stretchr/testify/require"
)

func TestOrchestration(t *testing.T) {

	t.Run("success json", func(t *testing.T) {
		var params orchestration.Parameters
		err := json.Unmarshal([]byte(`{"targets":{"include":[]}}`), &params)
		require.NoError(t, err)

		err = json.Unmarshal([]byte(`{"retryoperation": {}, "targets":{"include":[]}}`), &params)
		require.NoError(t, err)

		err = json.Unmarshal([]byte(`{"retryoperation": {"immediate": true}, "targets":{"include":[]}}`), &params)
		require.NoError(t, err)

		err = json.Unmarshal([]byte(`{"retryoperation": {"immediate": "true"}, "targets":{"include":[]}}`), &params)
		require.NoError(t, err)
	})

}
