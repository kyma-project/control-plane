package printer

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/events"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/liggitt/tabwriter"

	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/util/jsonpath"
)

const (
	tabwriterMinWidth = 6
	tabwriterWidth    = 4
	tabwriterPadding  = 3
	tabwriterPadChar  = ' '
	tabwriterFlags    = tabwriter.RememberWidths
)

type event struct {
	events.EventDTO
	Occurrence string
}

// newTabWriter returns a tabwriter that translates tabbed columns in input into properly aligned text.
func newTabWriter(output io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(output, tabwriterMinWidth, tabwriterWidth, tabwriterPadding, tabwriterPadChar, tabwriterFlags)
}

// FieldFormatterFunc is a function type to format and return the string representation of an object field.
type FieldFormatterFunc func(obj interface{}) string

// Column represents a user specified column.
// If FieldFormatter is not nil, it takes precedence over FieldSpec
type Column struct {
	Header string
	// FieldSpec is a JSONPath expression which specifies the field(s) to be printed
	FieldSpec string
	// FieldFormatter is a formatter fuction to print complex columns derived from object field(s).
	FieldFormatter FieldFormatterFunc
	parser         *jsonpath.JSONPath
}

// TablePrinter prints objects in table format, according to the given column definitions.
type TablePrinter interface {
	PrintObj(obj interface{}) error
	SetRuntimeEvents(eventList []events.EventDTO)
}

type tablePrinter struct {
	writer         *tabwriter.Writer
	columns        []Column
	events         map[string][]event
	eventsColumns  []Column
	noHeaders      bool
	headersPrinted bool
	now            time.Time
}

// NewTablePrinter creates a new TablePrinter.
// The parameter columns holds the non-empty list of Column specifications which comprises the table.
// If the parameter noHeaders is true, the first header row will not be displayed.
func NewTablePrinter(columns []Column, noHeaders bool) (TablePrinter, error) {
	t := &tablePrinter{
		writer:    newTabWriter(os.Stdout),
		columns:   columns,
		noHeaders: noHeaders,
		now:       time.Now(),
	}
	for idx := range t.columns {
		if t.columns[idx].FieldFormatter == nil && t.columns[idx].FieldSpec != "" {
			t.columns[idx].parser = jsonpath.New(fmt.Sprintf("column%d", idx)).AllowMissingKeys(true)
			if err := t.columns[idx].parser.Parse(t.columns[idx].FieldSpec); err != nil {
				return nil, err
			}
		}
	}

	return t, nil
}

func (t *tablePrinter) PrintObj(obj interface{}) error {
	defer t.writer.Flush()

	if !t.noHeaders && !t.headersPrinted {
		t.printHeader()
		t.headersPrinted = true
	}

	// Print the object, identify whether it is a slice of objects or single object
	s := reflect.ValueOf(obj)
	if s.Kind() == reflect.Slice {
		objs := toInterfaceSlice(obj)
		for idx := range objs {
			if err := t.printOneObj(objs[idx]); err != nil {
				return err
			}
		}
	} else {
		if err := t.printOneObj(obj); err != nil {
			return err
		}
	}

	return nil
}

func key(e events.EventDTO) string {
	iid := ""
	if e.InstanceID != nil {
		iid = *e.InstanceID
	}
	oid := ""
	if e.OperationID != nil {
		oid = *e.OperationID
	}
	return fmt.Sprintf("%v/%v/%v/%v", iid, oid, e.Level, e.Message)
}

func occurrence(now, start, end time.Time, count int) string {
	l := duration.HumanDuration(now.Sub(end))
	if count == 1 {
		return l
	}
	f := duration.HumanDuration(now.Sub(start))
	return fmt.Sprintf("%v ago (%vx in last %v)", l, count, f)
}

func (t *tablePrinter) deduplicateEvents(eventList []events.EventDTO) []event {
	m := make(map[string]int)
	start := make(map[string]time.Time)
	end := make(map[string]time.Time)
	for _, e := range eventList {
		k := key(e)
		m[k] += 1
		if start[k].IsZero() || start[k].After(e.CreatedAt) {
			start[k] = e.CreatedAt
		}
		if end[k].Before(e.CreatedAt) {
			end[k] = e.CreatedAt
		}
	}
	var events []event
	processed := make(map[string]int)
	for _, e := range eventList {
		k := key(e)
		count := m[k]
		processed[k] += 1
		pCount := processed[k]
		if count == pCount {
			event := event{
				EventDTO:   e,
				Occurrence: occurrence(t.now, start[k], end[k], count),
			}
			events = append(events, event)
		}
	}

	return events
}

func (t *tablePrinter) SetRuntimeEvents(eventList []events.EventDTO) {
	deduplicated := t.deduplicateEvents(eventList)
	t.events = make(map[string][]event)
	for _, e := range deduplicated {
		if e.InstanceID != nil {
			t.events[*e.InstanceID] = append(t.events[*e.InstanceID], e)
		}
	}
	t.eventsColumns = []Column{
		{
			Header:    "LEVEL",
			FieldSpec: "{.Level}",
		},
		{
			Header:    "OCCURRENCE",
			FieldSpec: "{.Occurrence}",
		},
		{
			Header:    "MESSAGE",
			FieldSpec: "{.Message}",
		},
	}
	for i := range t.eventsColumns {
		t.eventsColumns[i].parser = jsonpath.New(fmt.Sprintf("column%d", i)).AllowMissingKeys(true)
		t.eventsColumns[i].parser.Parse(t.eventsColumns[i].FieldSpec)
	}
}

func toInterfaceSlice(obj interface{}) []interface{} {
	s := reflect.ValueOf(obj)
	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

func (t *tablePrinter) printHeader() {
	for idx := range t.columns {
		fmt.Fprintf(t.writer, "%s\t", t.columns[idx].Header)
	}
	fmt.Fprint(t.writer, "\n")
}

func (t *tablePrinter) printOneObj(obj interface{}) error {
	for idx := range t.columns {
		if t.columns[idx].FieldFormatter != nil {
			fmt.Fprintf(t.writer, "%s\t", t.columns[idx].FieldFormatter(obj))
		} else {
			err := t.columns[idx].parser.Execute(t.writer, obj)
			fmt.Fprint(t.writer, "\t")
			if err != nil {
				return err
			}
		}
	}

	fmt.Fprintln(t.writer)
	if r, ok := obj.(runtime.RuntimeDTO); ok {
		if eventList, ok := t.events[r.InstanceID]; ok {
			if len(eventList) == 0 {
				return nil
			}
			lastOp := *eventList[0].OperationID
			buffer := strings.Builder{}
			eventTabWriter := newTabWriter(&buffer)
			printOperation(eventTabWriter, lastOp, r)
			for _, e := range eventList[:len(eventList)-1] {
				if lastOp != *e.OperationID {
					lastOp = *e.OperationID
					printOperation(eventTabWriter, lastOp, r)
				}
				if err := t.printEvent("˫", eventTabWriter, e); err != nil {
					return err
				}
			}
			if lastOp != *eventList[len(eventList)-1].OperationID {
				lastOp = *eventList[len(eventList)-1].OperationID
				printOperation(eventTabWriter, lastOp, r)
			}
			if err := t.printEvent("˪", eventTabWriter, eventList[len(eventList)-1]); err != nil {
				return err
			}
			fmt.Fprintln(eventTabWriter)
			eventTabWriter.Flush()
			t.writer.Write([]byte(buffer.String()))
		}
	}
	return nil
}

func (t *tablePrinter) printEvent(sep string, eventTabWriter io.Writer, e event) error {
	fmt.Fprintf(eventTabWriter, "  %v", sep)
	for _, col := range t.eventsColumns {
		if col.FieldFormatter != nil {
			fmt.Fprintf(eventTabWriter, "%s\t", col.FieldFormatter(e))
		} else {
			err := col.parser.Execute(eventTabWriter, e)
			fmt.Fprint(eventTabWriter, "\t")
			if err != nil {
				return err
			}
		}
	}
	fmt.Fprintln(eventTabWriter)
	return nil
}

func printOperation(w io.Writer, op string, rt runtime.RuntimeDTO) {
	if rt.Status.Provisioning != nil {
		if op == rt.Status.Provisioning.OperationID {
			opStatus := rt.Status.Provisioning.State
			fmt.Fprintf(w, " ˫%v operation %v: %v\n", "provision", op, opStatus)
			return
		}
	}
	if rt.Status.Deprovisioning != nil {
		if op == rt.Status.Deprovisioning.OperationID {
			opStatus := rt.Status.Deprovisioning.State
			fmt.Fprintf(w, " ˫%v operation %v: %v\n", "deprovision", op, opStatus)
			return
		}
	}
	if rt.Status.Update != nil {
		for _, update := range rt.Status.Update.Data {
			if op == update.OperationID {
				opStatus := update.State
				fmt.Fprintf(w, " ˫%v operation %v: %v\n", "update", op, opStatus)
				return
			}
		}
	}
	if rt.Status.UpgradingKyma != nil {
		for _, upgradeKyma := range rt.Status.UpgradingKyma.Data {
			if op == upgradeKyma.OperationID {
				opStatus := upgradeKyma.State
				fmt.Fprintf(w, " ˫%v operation %v: %v\n", "kyma upgrade", op, opStatus)
				return
			}
		}
	}
	if rt.Status.UpgradingCluster != nil {
		for _, upgradeCluster := range rt.Status.UpgradingCluster.Data {
			if op == upgradeCluster.OperationID {
				opStatus := upgradeCluster.State
				fmt.Fprintf(w, " ˫%v operation %v: %v\n", "cluster upgrade", op, opStatus)
				return
			}
		}
	}
	if rt.Status.Suspension != nil {
		for _, suspension := range rt.Status.Suspension.Data {
			if op == suspension.OperationID {
				opStatus := suspension.State
				fmt.Fprintf(w, " ˫%v operation %v: %v\n", "suspension", op, opStatus)
				return
			}
		}
	}
	if rt.Status.Unsuspension != nil {
		for _, unsuspension := range rt.Status.Unsuspension.Data {
			if op == unsuspension.OperationID {
				opStatus := unsuspension.State
				fmt.Fprintf(w, " ˫%v operation %v: %v\n", "unsuspension", op, opStatus)
				return
			}
		}
	}
	fmt.Fprintf(w, " ˫%v operation %v\n", "unknown", op)
}
