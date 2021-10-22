package command

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	mothershipClient "github.com/kyma-project/control-plane/components/reconciler/pkg/auth"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	ErrMothershipResponse = errors.New("reconciler error response")
)

//go:generate mockgen -destination=automock/keb_client.go -package=automock -source=reconciliations.go kebClient

type kebClient interface {
	ListRuntimes(runtime.ListParameters) (runtime.RuntimesPage, error)
}

type kebClientProvider = func(url string, httpClient *http.Client) kebClient

type mothershipClientProvider = func(url string, httpClient *http.Client) (mothership.ClientInterface, error)

type ReconciliationCommand struct {
	ctx         context.Context
	log         logger.Logger
	output      string
	rawStatuses []string
	runtimeIds  []string
	shoots      []string
	statuses    []mothership.Status

	provideKebClient   kebClientProvider
	provideMshipClient mothershipClientProvider
}

func toReconciliationStatuses(rawStates []string) ([]mothership.Status, error) {
	statuses := []mothership.Status{}
	if rawStates == nil {
		return nil, nil
	}
	for _, s := range rawStates {
		val := mothership.Status(strings.Trim(s, " "))
		switch val {
		case mothership.StatusReady, mothership.StatusError, mothership.StatusReconcilePending, mothership.StatusReconciling:
			statuses = append(statuses, val)
		default:
			return nil, fmt.Errorf("invalid value for state: %s", s)
		}
	}

	return statuses, nil
}

func (cmd *ReconciliationCommand) Validate() error {
	err := ValidateOutputOpt(cmd.output)
	if err != nil {
		return err
	}
	// Validate and transform states
	statuses, err := toReconciliationStatuses(cmd.rawStatuses)
	if err != nil {
		return err
	}
	cmd.statuses = statuses
	return nil
}

func (cmd *ReconciliationCommand) printReconciliation(data []mothership.Reconciliation) error {
	switch {
	case cmd.output == tableOutput:
		tp, err := printer.NewTablePrinter([]printer.Column{
			{
				Header:    "SCHEDULING ID",
				FieldSpec: "{.schedulingID}",
			},
			{
				Header:    "RUNTIME ID",
				FieldSpec: "{.runtimeID}",
			},
			{
				Header:         "CREATED AT",
				FieldSpec:      "{.created}",
				FieldFormatter: reconciliationCreated,
			},
			{
				Header:         "UPDATED",
				FieldSpec:      "{.updated}",
				FieldFormatter: reconciliationUpdated,
			},
			{
				Header:    "STATUES",
				FieldSpec: "{.status}",
			},
			{
				Header:    "LOCK",
				FieldSpec: "{.lock}",
			},
		}, false)
		if err != nil {
			return err
		}
		return tp.PrintObj(data)
	case cmd.output == jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		jp.PrintObj(data)
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
		return ccp.PrintObj(data)
	}
	return nil
}

func reconciliationCreated(obj interface{}) string {
	sr := obj.(mothership.Reconciliation)
	return sr.Created.Format("2006/01/02 15:04:05")
}

func reconciliationUpdated(obj interface{}) string {
	sr := obj.(mothership.Reconciliation)
	return sr.Updated.Format("2006/01/02 15:04:05")
}

func isErrResponse(statusCode int) bool {
	return statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices
}

func responseErr(resp *http.Response) error {
	msg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg = []byte(errors.Wrap(err, "unexpected error").Error())
	}
	return errors.Wrapf(ErrMothershipResponse, "%v %d", msg, resp.StatusCode)
}

func (cmd *ReconciliationCommand) Run() error {
	cmd.log = logger.New()

	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	auth := CLICredentialManager(cmd.log)
	httpClient := oauth2.NewClient(ctx, auth)

	runtimes := append([]string{}, cmd.runtimeIds...)
	// fetch runtime ids for all shoot names
	if len(cmd.shoots) > 0 {
		kebClient := cmd.provideKebClient(GlobalOpts.KEBAPIURL(), httpClient)

		listRtResp, err := kebClient.ListRuntimes(runtime.ListParameters{Shoots: cmd.shoots})
		if err != nil {
			return errors.Wrap(err, "while listing runtimes")
		}

		runtimes = append([]string{}, cmd.runtimeIds...)
		for _, dto := range listRtResp.Data {
			runtimes = append(runtimes, dto.RuntimeID)
		}
	}

	// fetch reconciliations
	mothershipURL := GlobalOpts.MothershipAPIURL()
	
	client, err := cmd.provideMshipClient(mothershipURL, httpClient)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	params := newGetReconciliationsParams(runtimes, cmd.statuses)
	response, err := client.GetReconciliations(ctx, params)
	if err != nil {
		return errors.Wrap(err, "wile listing reconciliations")
	}

	defer response.Body.Close()

	if isErrResponse(response.StatusCode) {
		err := responseErr(response)
		return err
	}

	var result []mothership.Reconciliation
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return errors.WithStack(ErrMothershipResponse)
	}

	err = cmd.printReconciliation(result)
	if err != nil {
		return errors.Wrap(err, "while printing runtimes")
	}

	return nil
}

func newReconciliationCmd(kp kebClientProvider, mp mothershipClientProvider) *cobra.Command {
	cmd := ReconciliationCommand{
		provideKebClient:   kp,
		provideMshipClient: mp,
	}

	cobraCmd := &cobra.Command{
		Use:     "reconciliations",
		Aliases: []string{"rc"},
		Short:   "Displays Kyma Reconciliations.",
		Long: `Displays Kyma Reconciliations and their primary attributes, such as reconciliation-id.
The command supports filtering Reconciliations based on`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.AddCommand(
		NewReconciliationEnableCmd(),
		NewReconciliationDisableCmd(),
		NewReconciliationOperationInfoCmd(),
	)

	SetOutputOpt(cobraCmd, &cmd.output)

	cobraCmd.Flags().StringSliceVarP(&cmd.runtimeIds, "runtime-id", "r", nil, "Filter by Runtime ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.rawStatuses, "status", "S", nil, "Filter by Reconciliation state. The possible values are: ok, err, suspended, all. Suspended Reconciliations are filtered out unless the \"all\" or \"suspended\" values are provided. You can provide multiple values, either separated by a comma (e.g. ok,err), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.shoots, "shoot", "c", nil, "Filter by Shoot cluster name. You can provide multiple values, either separated by a comma (e.g. shoot1,shoot2), or by specifying the option multiple times.")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
		return cobraCmd
	}

	cmd.ctx = context.Background()
	return cobraCmd
}

var (
	defaultKebClientProvider kebClientProvider = func(url string, httpClient *http.Client) kebClient {
		return runtime.NewClient(url, httpClient)
	}

	defaultMothershipClientProvider mothershipClientProvider = func(url string, httpClient *http.Client) (mothership.ClientInterface, error) {
		return mothershipClient.NewClient(url, httpClient)
	}
)

// NewUpgradeCmd constructs the reconciliation command and all subcommands under the reconciliation command
func NewReconciliationCmd() *cobra.Command {
	return newReconciliationCmd(defaultKebClientProvider, defaultMothershipClientProvider)
}

func newGetReconciliationsParams(runtimes []string, statuses []mothership.Status) *mothership.GetReconciliationsParams {
	var runtimeParams *[]string
	var statusesParams *[]mothership.Status

	if len(runtimes) != 0 {
		runtimeParams = &runtimes
	}

	if len(statuses) != 0 {
		statusesParams = &statuses
	}

	return &mothership.GetReconciliationsParams{
		RuntimeID: runtimeParams,
		Status:    statusesParams,
	}
}
