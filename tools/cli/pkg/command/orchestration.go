package command

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/spf13/cobra"
)

const (
	cancelCommand     = "cancel"
	operationsCommand = "operations"
	opsCommand        = "ops"
)

// OrchestrationCommand represents an execution of the kcp orchestrations command
type OrchestrationCommand struct {
	cobraCmd   *cobra.Command
	log        logger.Logger
	client     orchestration.Client
	output     string
	states     []string
	operation  string
	subCommand string
	listParams orchestration.ListParameters
}

var cliStates = map[string]string{
	"pending":    orchestration.Pending,
	"failed":     orchestration.Failed,
	"succeeded":  orchestration.Succeeded,
	"inprogress": orchestration.InProgress,
	"canceled":   orchestration.Canceled,
	"canceling":  orchestration.Canceling,
}

var orchestrationColumns = []printer.Column{
	{
		Header:    "ORCHESTRATION ID",
		FieldSpec: "{.OrchestrationID}",
	},
	{
		Header:         "TYPE",
		FieldFormatter: orchestrationType,
	},
	{
		Header:         "CREATED AT",
		FieldFormatter: orchestrationCreatedAt,
	},
	{
		Header:    "STATE",
		FieldSpec: "{.State}",
	},
	{
		Header:    "DRY RUN",
		FieldSpec: "{.Parameters.DryRun}",
	},
	{
		Header:         "TARGETS",
		FieldFormatter: orchestrationTargets,
	},
	{
		Header:         "DETAILS",
		FieldFormatter: orchestrationDetails,
	},
}

var operationColumns = []printer.Column{
	{
		Header:    "OPERATION ID",
		FieldSpec: "{.OperationID}",
	},
	{
		Header:    "SHOOT",
		FieldSpec: "{.ShootName}",
	},
	{
		Header:    "GLOBALACCOUNT",
		FieldSpec: "{.GlobalAccountID}",
	},
	{
		Header:    "SUBACCOUNT",
		FieldSpec: "{.SubAccountID}",
	},
	{
		Header:    "STATE",
		FieldSpec: "{.State}",
	},
}

var orchestrationDetailsTpl = `Orchestration ID: {{.OrchestrationID}}
Type:             {{.Type}}
Created At:       {{.CreatedAt}}
Updated At:       {{.UpdatedAt}}
Dry Run:          {{.Parameters.DryRun}}
State:            {{.State}}
Description:      {{.Description}}
Strategy:         {{.Parameters.Strategy.Type}}
Schedule:         {{.Parameters.Strategy.Schedule}}
Workers:          {{.Parameters.Strategy.Parallel.Workers}}
Targets:
{{- range $i, $t := .Parameters.Targets.Include }}
  {{ orchestrationTarget $t }}
{{- end -}}
{{- if gt (len .Parameters.Targets.Exclude) 0 }}
Exclude Targets:
{{- range $i, $t := .Parameters.Targets.Exclude }}
  {{ orchestrationTarget $t }}
{{- end -}}
{{- end -}}
{{- if gt (len .OperationStats) 0 }}
Operations:
{{- range $i, $s := orchestrationStates }}
{{- if gt (index $.OperationStats $s) 0 }}
  {{ printf "%11s: %d" $s (index $.OperationStats $s) }}
{{- end -}}
{{- end -}}
{{- end }}
`

var operationDetailsTpl = `Operation ID:       {{.OperationID}}
Orchestration ID:   {{.OrchestrationID}}
Global Account ID:  {{.GlobalAccountID}}
Subaccount ID:      {{.SubAccountID}}
Runtime ID:         {{.RuntimeID}}
Shoot Name:         {{.ShootName}}
Service Plan:       {{.ServicePlanName}}
Maintenance Window: {{.MaintenanceWindowBegin}} - {{.MaintenanceWindowEnd}}
State:              {{.State}}
Description:        {{.Description}}
Kubernetes Version: {{.ClusterConfig.KubernetesVersion}}
Kyma Version:       {{.KymaConfig.Version}}
`

// NewOrchestrationCmd constructs a new instance of OrchestrationCommand and configures it in terms of a cobra.Command
func NewOrchestrationCmd() *cobra.Command {
	cmd := OrchestrationCommand{}
	cobraCmd := &cobra.Command{
		Use:     "orchestrations [id] [ops|operations] [cancel]",
		Aliases: []string{"orchestration", "o"},
		Short:   "Displays Kyma Control Plane (KCP) orchestrations.",
		Long: `Displays KCP orchestrations and their primary attributes, such as identifiers, type, state, parameters, or Runtime operations.
The command has the following modes:
  - Without specifying an orchestration ID as an argument. In this mode, the command lists all orchestrations, or orchestrations matching the --state option, if provided.
  - When specifying an orchestration ID as an argument. In this mode, the command displays details about the specific orchestration.
      If the optional --operation flag is provided, it displays details of the specified Runtime operation within the orchestration.
  - When specifying an orchestration ID and ` + "`operations` or `ops`" + ` as arguments. In this mode, the command displays the Runtime operations for the given orchestration.
  - When specifying an orchestration ID and ` + "`cancel`" + ` as arguments. In this mode, the command cancels the orchestration and all pending Runtime operations.`,
		Example: `  kcp orchestrations --state inprogress                                   Display all orchestrations which are in progress.
  kcp orchestration -o custom="Orchestration ID:{.OrchestrationID},STATE:{.State},CREATED AT:{.createdAt}"
                                                                          Display all orchestations with specific custom fields.
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00                  Display details about a specific orchestration.
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00 --operation OID  Display details of the specified Runtime operation within the orchestration.
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00 operations       Display the operations of the given orchestration.
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00 cancel           Cancel the given orchestration.`,
		Args:    cobra.MaximumNArgs(2),
		PreRunE: func(_ *cobra.Command, args []string) error { return cmd.Validate(args) },
		RunE:    func(_ *cobra.Command, args []string) error { return cmd.Run(args) },
	}
	cmd.cobraCmd = cobraCmd

	SetOutputOpt(cobraCmd, &cmd.output)
	cobraCmd.Flags().StringSliceVarP(&cmd.states, "state", "s", nil, fmt.Sprintf("Filter output by state. You can provide multiple values, either separated by a comma (e.g. failed,inprogress), or by specifying the option multiple times. The possible values are: %s.", strings.Join(cliOrchestrationStates(), ", ")))
	cobraCmd.Flags().StringVar(&cmd.operation, "operation", "", "Option that displays details of the specified Runtime operation when a given orchestration is selected.")
	return cobraCmd
}

func cliOrchestrationStates() []string {
	s := []string{}
	for state := range cliStates {
		s = append(s, state)
	}
	sort.Strings(s)

	return s
}

func orchestrationStates() []string {
	s := []string{}
	for _, state := range cliStates {
		s = append(s, state)
	}
	sort.Strings(s)

	return s
}

func (cmd *OrchestrationCommand) validateTransformOrchestrationStates() error {
	for _, inputState := range cmd.states {
		if state, ok := cliStates[inputState]; ok {
			cmd.listParams.States = append(cmd.listParams.States, state)
		} else {
			return fmt.Errorf("invalid value for state: %s", inputState)
		}
	}

	return nil
}

// Run executes the orchestrations command
func (cmd *OrchestrationCommand) Run(args []string) error {
	cmd.log = logger.New()
	cmd.client = orchestration.NewClient(cmd.cobraCmd.Context(), GlobalOpts.KEBAPIURL(), CLICredentialManager(cmd.log))

	switch len(args) {
	case 0:
		// Called without any arguments: list orchestrations
		return cmd.showOrchestrations()
	case 1:
		// Called with orchestration ID but without subcommand
		if cmd.operation == "" {
			return cmd.showOneOrchestration(args[0])
		}
		return cmd.showOperationDetails(args[0])
	case 2:
		// Called with orchestration ID and subcommand
		switch cmd.subCommand {
		case cancelCommand:
			return cmd.cancelOrchestration(args[0])
		case operationsCommand, opsCommand:
			return cmd.showOperations(args[0])
		}
	}

	return nil
}

// Validate checks the input parameters of the orchestrations command
func (cmd *OrchestrationCommand) Validate(args []string) error {
	err := ValidateOutputOpt(cmd.output)
	if err != nil {
		return err
	}

	err = cmd.validateTransformOrchestrationStates()
	if err != nil {
		return err
	}

	if cmd.operation != "" && len(args) == 0 {
		return errors.New("--operation should only be used when orchestration id is given as an argument")
	}
	if cmd.operation != "" && len(cmd.states) > 0 {
		return errors.New("--state should not be used together with --operation")
	}

	if len(args) == 2 {
		cmd.subCommand = args[1]
		switch cmd.subCommand {
		case cancelCommand, operationsCommand, opsCommand:
		default:
			return fmt.Errorf("invalid subcommand: %s", cmd.subCommand)
		}
	}

	return nil
}

func (cmd *OrchestrationCommand) showOrchestrations() error {
	srl, err := cmd.client.ListOrchestrations(cmd.listParams)
	if err != nil {
		return errors.Wrap(err, "while listing orchestrations")
	}

	switch {
	case cmd.output == tableOutput:
		tp, err := printer.NewTablePrinter(orchestrationColumns, false)
		if err != nil {
			return err
		}
		return tp.PrintObj(srl.Data)
	case cmd.output == jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		jp.PrintObj(srl)
	case strings.HasPrefix(cmd.output, customOutput):
		_, templateFile := printer.ParseOutputToTemplateTypeAndElement(cmd.output)
		column, err := printer.ParseColumnToHeaderAndFieldSpec(templateFile)
		if err != nil {
			return err
		}

		ccp, err := printer.NewTablePrinter(column, false)
		if err != nil {
			return err
		}
		return ccp.PrintObj(srl.Data)
	}
	return nil
}

func (cmd *OrchestrationCommand) showOneOrchestration(orchestrationID string) error {
	sr, err := cmd.client.GetOrchestration(orchestrationID)
	if err != nil {
		return errors.Wrap(err, "while getting orchestration")
	}

	switch cmd.output {
	case tableOutput:
		// Print orchestration details via template
		funcMap := template.FuncMap{
			"orchestrationTarget": orchestrationTarget,
			"orchestrationStates": orchestrationStates,
		}
		tmpl, err := template.New("orchestrationDetails").Funcs(funcMap).Parse(orchestrationDetailsTpl)
		if err != nil {
			return errors.Wrap(err, "while parsing orchestration details template")
		}
		err = tmpl.Execute(os.Stdout, sr)
		if err != nil {
			return errors.Wrap(err, "while printing orchestration details")
		}
	case jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		jp.PrintObj(sr)
	}

	return nil
}

func (cmd *OrchestrationCommand) showOperations(orchestrationID string) error {
	orl, err := cmd.client.ListOperations(orchestrationID, cmd.listParams)
	if err != nil {
		return errors.Wrap(err, "while listing operations")
	}

	switch cmd.output {
	case tableOutput:
		// Print operation table
		if len(orl.Data) > 0 {
			tp, err := printer.NewTablePrinter(operationColumns, false)
			if err != nil {
				return err
			}
			return tp.PrintObj(orl.Data)
		}
	case jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		jp.PrintObj(orl)
	}

	return nil
}

func (cmd *OrchestrationCommand) showOperationDetails(orchestrationID string) error {
	odr, err := cmd.client.GetOperation(orchestrationID, cmd.operation)
	if err != nil {
		return errors.Wrap(err, "while getting operation details")
	}

	switch cmd.output {
	case tableOutput:
		tmpl, err := template.New("operationDetails").Parse(operationDetailsTpl)
		if err != nil {
			return errors.Wrap(err, "while parsing operation details template")
		}
		err = tmpl.Execute(os.Stdout, odr)
		if err != nil {
			return errors.Wrap(err, "while printing operation details")
		}
	case jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		jp.PrintObj(odr)
	}

	return nil
}

func (cmd *OrchestrationCommand) cancelOrchestration(orchestrationID string) error {
	sr, err := cmd.client.GetOrchestration(orchestrationID)
	if err != nil {
		return errors.Wrap(err, "while getting orchestration")
	}
	switch sr.State {
	case orchestration.Canceling, orchestration.Canceled:
		fmt.Println("Orchestration is already canceled.")
		return nil
	case orchestration.Failed, orchestration.Succeeded:
		return fmt.Errorf("orchestration is already %s", sr.State)
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("%d pending operations(s) will be canceled, %d in progress operation(s) will still be completed.\n", sr.OperationStats[orchestration.Pending], sr.OperationStats[orchestration.InProgress])
	fmt.Print("Do you want to continue? (Y/N) ")
	scanner.Scan()
	if scanner.Text() != "Y" {
		fmt.Println("Aborted.")
		return nil
	}

	return cmd.client.CancelOrchestration(orchestrationID)

}

// Currently only orchestrations of type "kyma upgrade" are supported,
// and the type is not reflected in the StatusResponse object
func orchestrationType(obj interface{}) string {
	sr := obj.(orchestration.StatusResponse)

	if sr.Type == orchestration.UpgradeKymaOrchestration {
		return "kyma upgrade"
	}
	if sr.Type == orchestration.UpgradeClusterOrchestration {
		return "cluster upgrade"
	}
	return string(sr.Type)
}

func orchestrationCreatedAt(obj interface{}) string {
	sr := obj.(orchestration.StatusResponse)
	return sr.CreatedAt.Format("2006/01/02 15:04:05")
}

// orchestrationTarget returns the string representation of a orchestration.RuntimeTarget
func orchestrationTarget(t orchestration.RuntimeTarget) string {
	targets := []string{}
	if t.Target != "" {
		targets = append(targets, fmt.Sprintf("target = %s", t.Target))
	}
	if t.GlobalAccount != "" {
		targets = append(targets, fmt.Sprintf("account = %s", t.GlobalAccount))
	}
	if t.SubAccount != "" {
		targets = append(targets, fmt.Sprintf("subaccount = %s", t.SubAccount))
	}
	if t.RuntimeID != "" {
		targets = append(targets, fmt.Sprintf("runtime-id = %s", t.RuntimeID))
	}
	if t.InstanceID != "" {
		targets = append(targets, fmt.Sprintf("instance-id = %s", t.InstanceID))
	}
	if t.Region != "" {
		targets = append(targets, fmt.Sprintf("region = %s", t.Region))
	}
	if t.PlanName != "" {
		targets = append(targets, fmt.Sprintf("plan = %s", t.PlanName))
	}
	if t.Shoot != "" {
		targets = append(targets, fmt.Sprintf("shoot = %s", t.Shoot))
	}

	return strings.Join(targets, ",")
}

// orchestrationTarget returns the string representation of an array of orchestration.RuntimeTarget
func orchestrationTargets(obj interface{}) string {
	sr := obj.(orchestration.StatusResponse)
	var sb strings.Builder
	nTargets := len(sr.Parameters.Targets.Include)
	for i := 0; i < nTargets; i++ {
		runtimeTarget := sr.Parameters.Targets.Include[i]
		if runtimeTarget.Target != "" {
			sb.WriteString(runtimeTarget.Target)
		} else if runtimeTarget.GlobalAccount != "" {
			sb.WriteString("GlobalAccount=" + runtimeTarget.GlobalAccount)
		} else if runtimeTarget.SubAccount != "" {
			sb.WriteString("SubAccount=" + runtimeTarget.SubAccount)
		} else if runtimeTarget.Region != "" {
			sb.WriteString("Region=" + runtimeTarget.Region)
		} else if runtimeTarget.RuntimeID != "" {
			sb.WriteString("RuntimeID=" + runtimeTarget.RuntimeID)
		} else if runtimeTarget.PlanName != "" {
			sb.WriteString("PlanName=" + runtimeTarget.PlanName)
		} else if runtimeTarget.Shoot != "" {
			sb.WriteString("Shoot=" + runtimeTarget.Shoot)
		} else if runtimeTarget.InstanceID != "" {
			sb.WriteString("InstanceID=" + runtimeTarget.InstanceID)
		}
		if i != (nTargets - 1) {
			sb.WriteString(", ")
		}
	}

	// Limit the targets to 20 characters
	targets := sb.String()
	if len(targets) > 20 {
		targets = targets[0:20]
	}
	return targets
}

func orchestrationDetails(obj interface{}) string {
	sr := obj.(orchestration.StatusResponse)
	var sb strings.Builder

	if sr.Type == orchestration.UpgradeKymaOrchestration {
		if sr.Parameters.Kyma != nil && sr.Parameters.Kyma.Version != "" {
			sb.WriteString("Kyma: " + sr.Parameters.Kyma.Version)
		}
	} else if sr.Type == orchestration.UpgradeClusterOrchestration {
		if sr.Parameters.Kubernetes != nil && sr.Parameters.Kubernetes.KubernetesVersion != "" {
			sb.WriteString("K8S: " + sr.Parameters.Kubernetes.KubernetesVersion)
		}
	} else {
		sb.WriteString("-")
	}

	return sb.String()
}
