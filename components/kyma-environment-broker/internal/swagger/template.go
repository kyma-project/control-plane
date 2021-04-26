package swagger

import (
	"os"
	"text/template"

	"github.com/pkg/errors"
)

type SchemaProvider interface {
	Execute() error
}

type Template struct {
	SwaggerFilesPath string
	Templates        map[string]string
}

func NewTemplate(swaggerFilesPath string, templates map[string]string) *Template {
	return &Template{
		SwaggerFilesPath: swaggerFilesPath,
		Templates:        templates,
	}
}

func (t *Template) Execute() error {
	templateSchemaPath := t.SwaggerFilesPath + "/schema/swagger.yaml"
	// this path is also set in the files/swagger/index.html file
	outputSchemaPath := t.SwaggerFilesPath + "/swagger.yaml"

	schema, err := template.ParseFiles(templateSchemaPath)
	if err != nil {
		return errors.Wrap(err, "while parsing files")
	}
	output, err := os.OpenFile(outputSchemaPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return errors.Wrap(err, "while opening a file")
	}
	defer output.Close()

	err = schema.Execute(output, t.Templates)
	if err != nil {
		return errors.Wrap(err, "while executing template")
	}
	return nil
}
