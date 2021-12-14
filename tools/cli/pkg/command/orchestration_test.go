package command

import (
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/stretchr/testify/require"
)

func TestShowOperationDetailsOutput(t *testing.T) {
	fixID := "orchestration_id_0"
	cmd := fixOrchestrationCommand()

	odrs := fixOperationDetailResponse(cmd.operations, fixID)

	switch cmd.output {
	case tableOutput:
		tmpl, err := template.New("operationDetails").Parse(operationsDetailsTpl)
		require.NoError(t, err)

		err = tmpl.Execute(os.Stdout, odrs)
		require.NoError(t, err)

	case jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		jp.PrintObj(odrs)
	}

}

func TestRetryOrchestrationOutput(t *testing.T) {
	fixID := "orchestration_id_0"
	rr := fixRetryOrchestrationResponse(fixID)

	funcMap := template.FuncMap{
		"stringsJoin": strings.Join,
	}
	tmpl, err := template.New("retryOrchestration").Funcs(funcMap).Parse(retryOchestrationTpl)
	require.NoError(t, err)

	err = tmpl.Execute(os.Stdout, rr)
	require.NoError(t, err)

}

func fixOrchestrationCommand() *OrchestrationCommand {
	cmd := OrchestrationCommand{}
	cmd.operations = []string{"operation_id_0", "operation_id_1"}
	cmd.output = jsonOutput

	return &cmd

}

func fixOperationResponse(id, orchestrationID string) orchestration.OperationResponse {
	return orchestration.OperationResponse{
		OperationID:     id,
		RuntimeID:       "runtime_id" + id,
		OrchestrationID: orchestrationID,
	}
}

func fixOperationDetailResponse(ids []string, orchestrationID string) []orchestration.OperationDetailResponse {
	odrs := []orchestration.OperationDetailResponse{}

	for _, id := range ids {
		odrs = append(odrs, orchestration.OperationDetailResponse{
			OperationResponse: fixOperationResponse(id, orchestrationID),
		})
	}
	return odrs
}

func fixRetryOrchestrationResponse(orchestrationID string) orchestration.RetryResponse {
	return orchestration.RetryResponse{
		OrchestrationID:   orchestrationID,
		RetryOperations:   []string{"id-2"},
		OldOperations:     []string{"id-0"},
		InvalidOperations: []string{"id-1", "id-3", "id-10"},
		Msg:               "retry operations are queued for processing",
	}
}
