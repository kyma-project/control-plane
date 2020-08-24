package templates

import (
	"bytes"
	"fmt"
	"text/template"
)

type Values struct {
	ShootName          string
	ProjectName        string
	GardenerSecretName string
	Region             string
}

// TODO: test
func RenderTemplate(text string, values Values) ([]byte, error) {
	templ, err := template.New("").Parse(text)
	if err != nil {
		return nil, fmt.Errorf("error while parsing renderTemplate: %s", err.Error())
	}

	buffer := new(bytes.Buffer)
	err = templ.Execute(buffer, values)
	if err != nil {
		return nil, fmt.Errorf("error while rendering renderTemplate: %s", err.Error())
	}

	return buffer.Bytes(), nil
}
