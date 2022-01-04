package command

import (
	"context"
	"encoding/json"
	"strings"

	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type RuntimeStateCommand struct {
	output        string
	runtimeID     string
	correlationID string
	schedulingID  string

	ctx context.Context

	provideMshipClient mothershipClientProvider
}

func NewRuntimeStateCommand() *cobra.Command {
	return newRuntimeStateCommand(defaultMothershipClientProvider)
}

func newRuntimeStateCommand(mp mothershipClientProvider) *cobra.Command {
	cmd := RuntimeStateCommand{
		provideMshipClient: mp,
	}
	cobraCmd := &cobra.Command{
		Use:     "state",
		Aliases: []string{"s"},
		Short:   "",
		Long:    ``,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	SetOutputOpt(cobraCmd, &cmd.output)
	cobraCmd.Flags().StringVarP(&cmd.runtimeID, "runtime-id", "r", "", "")
	cobraCmd.Flags().StringVarP(&cmd.correlationID, "correlation-id", "c", "", "")
	cobraCmd.Flags().StringVarP(&cmd.schedulingID, "scheduling-id", "s", "", "")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
		return cobraCmd
	}

	cmd.ctx = context.Background()
	return cobraCmd
}

func (cmd *RuntimeStateCommand) Validate() error {
	count := 0
	if cmd.correlationID != "" {
		count++
	}
	if cmd.runtimeID != "" {
		count++
	}
	if cmd.schedulingID != "" {
		count++
	}

	if count != 1 {
		return errors.New("use one of following flags: --runtime-id, --correlation-id or --scheduling-id")
	}

	return ValidateOutputOpt(cmd.output)
}

func (cmd *RuntimeStateCommand) Run() error {
	l := logger.New()
	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	auth := CLICredentialManager(l)
	httpClient := oauth2.NewClient(ctx, auth)

	mothershipURL := GlobalOpts.MothershipAPIURL()

	client, err := cmd.provideMshipClient(mothershipURL, httpClient)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	response, err := client.GetClustersState(ctx, &mothership.GetClustersStateParams{
		RuntimeID:     &cmd.runtimeID,
		SchedulingID:  &cmd.schedulingID,
		CorrelationID: &cmd.correlationID,
	})
	if err != nil {
		return errors.Wrap(err, "wile listing reconciliations")
	}

	defer response.Body.Close()

	if isErrResponse(response.StatusCode) {
		err := responseErr(response)
		return err
	}

	var result mothership.HTTPClusterState
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return errors.WithStack(ErrMothershipResponse)
	}

	return printState(cmd.output, result)
}

func printState(format string, data mothership.HTTPClusterState) error {
	switch {
	case format == tableOutput:
		tp, err := printer.NewTablePrinter([]printer.Column{
			{
				Header:    "RUNTIME ID",
				FieldSpec: "{.cluster.runtimeID}",
			},
			{
				Header:         "CREATED AT",
				FieldSpec:      "{.cluster.created}",
				FieldFormatter: reconciliationCreated,
			},
		}, false)
		if err != nil {
			return err
		}

		return tp.PrintObj(data)
	case format == jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		return jp.PrintObj(data)
	case strings.HasPrefix(format, customOutput):
		_, templateFile := printer.ParseOutputToTemplateTypeAndElement(format)
		column, err := printer.ParseColumnToHeaderAndFieldSpec(templateFile)
		if err != nil {
			return err
		}

		ccp, err := printer.NewTablePrinter(column, false)
		if err != nil {
			return err
		}
		return ccp.PrintObj(data)
	default:
		return errors.Errorf("unknown output format: %s", format)
	}
}
