package swagger

import (
	"fmt"
	"os"
	"text/template"
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
		return fmt.Errorf("while parsing files: %w", err)
	}
	output, err := os.OpenFile(outputSchemaPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return fmt.Errorf("while opening file: %w", err)
	}

	defer output.Close()

	err = schema.Execute(output, t.Templates)
	if err != nil {
		return fmt.Errorf("while executing template: %w", err)
	}
	return nil
}
