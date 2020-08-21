package main

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/provisioner/internal/templates"
	log "github.com/sirupsen/logrus"

	"io/ioutil"
	"os"
)

const (
	shootTemplatePath = "templates/shoot.yaml"
)

type ShootValues struct {
	ShootName string
	ProjectName string
	GardenerSecretName string
	Region string
}

func main() {

	// TODO: opts


	err := generateShootTemplate()
	exitOnError(err, "failed to generate Shoot template")

}

func generateShootTemplate() error {
	shootTemplate, err := templates.GenerateShootTemplate()
	if err != nil {
		return fmt.Errorf("error when generating Shoot tamplate: %s", err.Error())
	}

	err = ioutil.WriteFile(shootTemplatePath, shootTemplate, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error when writing template to file: %s", err.Error())
	}

	return nil
}

func exitOnError(err error, context string) {
	if err != nil {
		wrappedError := fmt.Errorf("%s: %s", context, err.Error())
		log.Fatal(wrappedError)
	}
}

