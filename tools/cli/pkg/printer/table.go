package printer

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/liggitt/tabwriter"
	"k8s.io/client-go/util/jsonpath"
)

const (
	tabwriterMinWidth = 6
	tabwriterWidth    = 4
	tabwriterPadding  = 3
	tabwriterPadChar  = ' '
	tabwriterFlags    = tabwriter.RememberWidths
)

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
}

type tablePrinter struct {
	writer         *tabwriter.Writer
	columns        []Column
	noHeaders      bool
	headersPrinted bool
}

// NewTablePrinter creates a new TablePrinter.
// The parameter columns holds the non-empty list of Column specifications which comprises the table.
// If the parameter noHeaders is true, the first header row will not be displayed.
func NewTablePrinter(columns []Column, noHeaders bool) (TablePrinter, error) {
	t := &tablePrinter{
		writer:    newTabWriter(os.Stdout),
		columns:   columns,
		noHeaders: noHeaders,
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

	fmt.Fprint(t.writer, "\n")
	return nil
}
