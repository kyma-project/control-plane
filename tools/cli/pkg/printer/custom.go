package printer

import (
	"fmt"
	"regexp"
	"strings"
)

var jsonRegexp = regexp.MustCompile(`^\{\.?([^{}]+)\}$|^\.?([^{}]+)$`)
var templateFormat = []string{"custom="}

// RelaxedJSONPathExpression attempts to be flexible with JSONPath expressions, it accepts:
//   * metadata.name (no leading '.' or curly braces '{...}'
//   * {metadata.name} (no leading '.')
//   * .metadata.name (no curly braces '{...}')
//   * {.metadata.name} (complete expression)
// And transforms them all into a valid jsonpath expression:
//   {.metadata.name}
func RelaxedJSONPathExpression(pathExpression string) (string, error) {
	if len(pathExpression) == 0 {
		return pathExpression, nil
	}
	submatches := jsonRegexp.FindStringSubmatch(pathExpression)
	if submatches == nil {
		return "", fmt.Errorf("unexpected path string, expected a 'name1.name2' or '.name1.name2' or '{name1.name2}' or '{.name1.name2}'")
	}
	if len(submatches) != 3 {
		return "", fmt.Errorf("unexpected submatch list: %v", submatches)
	}
	var fieldSpec string
	if len(submatches[1]) != 0 {
		fieldSpec = submatches[1]
	} else {
		fieldSpec = submatches[2]
	}
	return fmt.Sprintf("{.%s}", fieldSpec), nil
}

//ParseOutputToTemplateTypeAndElement parses the output into templateType and templateElement
//e.g. kcp runtimes  -o custom="INSTANCE ID:instanceID,SHOOTNAME:shootName"
//After parsing, the templateType = "custom" and  templateElement = "INSTANCE ID:instanceID,SHOOTNAME:shootName"
func ParseOutputToTemplateTypeAndElement(output string) (string, string) {
	var templateType, templateElement string
	for _, format := range templateFormat {
		if strings.HasPrefix(output, format) {
			templateElement = output[len(format):]
			templateType = format[:len(format)-1]
		}
	}
	return templateType, templateElement
}

// ParseColumnToHeaderAndFieldSpec parses a custom columns contents to a list of Column <header>:<jsonpath-field-spec> pairs.
// e.g. spec is INSTANCE ID:instanceID,SHOOTNAME:shootName
// columnsOut[0].Header = "INSTANCE ID" and  columnsOut[0].FieldSpec = "{.instanceID}"
// columnsOut[1].Header = "SHOOTNAME"   and  columnsOut[1].FieldSpec = "{.shootName}"
func ParseColumnToHeaderAndFieldSpec(spec string) ([]Column, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("custom format specified but no custom columns given")
	}
	parts := strings.Split(spec, ",")
	columnsOut := make([]Column, len(parts))
	for ix := range parts {
		colSpec := strings.SplitN(parts[ix], ":", 2)
		if len(colSpec) != 2 {
			return nil, fmt.Errorf("unexpected custom spec: %s, expected <header>:<json-path-expr>", parts[ix])
		}
		spec, err := RelaxedJSONPathExpression(colSpec[1])
		if err != nil {
			return nil, err
		}
		columnsOut[ix] = Column{Header: colSpec[0], FieldSpec: spec}
	}
	return columnsOut, nil
}
