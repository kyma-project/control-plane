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

type RuntimeStateOptions struct {
	output        string
	runtimeID     string
	shootName     string
	correlationID string
	schedulingID  string
}

type RuntimeStateCommand struct {
	opts RuntimeStateOptions
	ctx  context.Context

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
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.opts.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	SetOutputOpt(cobraCmd, &cmd.opts.output)
	cobraCmd.Flags().StringVarP(&cmd.opts.runtimeID, "runtime-id", "r", "", "Runtime ID of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVarP(&cmd.opts.shootName, "shoot", "c", "", "Shoot cluster name of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVar(&cmd.opts.correlationID, "correlation-id", "", "Correlation ID of the specific Reconciliation Operation.")
	cobraCmd.Flags().StringVar(&cmd.opts.schedulingID, "scheduling-id", "", "Scheduling ID of the specific Reconciliation Operation.")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
		return cobraCmd
	}

	cmd.ctx = context.Background()
	return cobraCmd
}

func (opts *RuntimeStateOptions) Validate() error {
	count := 0
	if opts.correlationID != "" {
		count++
	}
	if opts.runtimeID != "" {
		count++
	}
	if opts.schedulingID != "" {
		count++
	}
	if opts.shootName != "" {
		count++
	}

	if count != 1 {
		return errors.New("use one of following flags: --shoot, --runtime-id, --correlation-id or --scheduling-id")
	}

	return ValidateOutputOpt(opts.output)
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

	runtimeID := cmd.opts.runtimeID
	if cmd.opts.shootName != "" {
		kebURL := GlobalOpts.KEBAPIURL()
		runtimeID, err = getRuntimeID(ctx, kebURL, cmd.opts.shootName, httpClient)
		if err != nil {
			return errors.Wrap(err, "while getting runtime ID")
		}

	}

	response, err := client.GetClustersState(ctx, &mothership.GetClustersStateParams{
		RuntimeID:     &runtimeID,
		SchedulingID:  &cmd.opts.schedulingID,
		CorrelationID: &cmd.opts.correlationID,
	})
	if err != nil {
		return errors.Wrap(err, "wile getting cluster state")
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

	return printState(cmd.opts.output, result)
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
				Header:    "KYMA VERSION",
				FieldSpec: "{.configuration.kymaVersion}",
			},
			{
				Header:    "KYMA PROFILE",
				FieldSpec: "{.configuration.kymaProfile}",
			},
			{
				Header:    "STATUS",
				FieldSpec: "{.status.status}",
			},
			{
				Header:    "DELETED",
				FieldSpec: "{.status.deleted}",
			},
			{
				Header:         "CREATED AT",
				FieldSpec:      "{.status.created}",
				FieldFormatter: stateCreatedFormatted,
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

func stateCreatedFormatted(obj interface{}) string {
	state := obj.(mothership.HTTPClusterState)
	return state.Cluster.Created.Format("2006/01/02 15:04:05")
}
