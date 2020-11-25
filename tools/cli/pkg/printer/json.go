package printer

import (
	"os"

	"encoding/json"
)

// JSONPrinter prints objects in JSON format
type JSONPrinter interface {
	PrintObj(obj interface{}) error
}

type jsonPrinter struct {
	e *json.Encoder
}

// NewJSONPrinter creates a new JSONPrinter.
// If indent is set to a non-empty string, the output will be pretty-printed, and the specified string will be applied for each level of indentation.
func NewJSONPrinter(indent string) JSONPrinter {
	j := &jsonPrinter{
		e: json.NewEncoder(os.Stdout),
	}
	if indent != "" {
		j.e.SetIndent("", indent)
	}
	return j
}

func (j jsonPrinter) PrintObj(obj interface{}) error {
	return j.e.Encode(obj)
}
