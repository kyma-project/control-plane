package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/kyma-project/control-plane/components/provisioner/internal/templates"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	templatesDir      = "templates"
	shootTemplateName = "shoot.yaml"
)

func main() {
	var app = cli.NewApp()

	app.Commands = []*cli.Command{
		{
			Name:    "generate",
			Aliases: []string{"gen"},
			Usage:   "Generate Shoot template",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "provider",
					Value: "azure",
					Usage: "Underlying cloud provider for Gardener to use",
				},
				&cli.PathFlag{
					Name:    "out",
					Aliases: []string{"o"},
					Value:   path.Join(templatesDir, shootTemplateName),
					Usage:   "Output file to which Shoot template will be saved",
				},
			},
			Action: func(c *cli.Context) error {
				outPath := c.Path("out")
				fmt.Printf("\nGenerating Shoot template in '%s'...\n", outPath)
				return generateShootTemplate(c.String("provider"), outPath)
			},
		},
		{
			Name:  "render",
			Usage: "Render templates with provided values",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "shoot",
					Value: "my-shoot",
					Usage: "Name of the Shoot",
				},
				&cli.StringFlag{
					Name:     "project",
					Required: true,
					Usage:    "Name of the Gardener project",
				},
				&cli.StringFlag{
					Name:     "secret",
					Required: true,
					Usage:    "Name of the Gardener secret",
				},
				&cli.StringFlag{
					Name:  "region",
					Value: "westeurope",
					Usage: "Region in which cluster should be deployed. One of: northeurope, westeurope, centralus, westus2",
				},
				&cli.PathFlag{
					Name:  "dir",
					Value: templatesDir,
					Usage: "Directory containing the templates",
				},
				&cli.PathFlag{
					Name:    "out",
					Aliases: []string{"o"},
					Value:   "templates-rendered",
					Usage:   "Output directory to which resources will be rendered",
				},
			},
			Action: func(c *cli.Context) error {
				values := templates.Values{
					ShootName:          c.String("shoot"),
					ProjectName:        c.String("project"),
					GardenerSecretName: c.String("secret"),
					Region:             c.String("region"),
				}

				inPath := c.Path("dir")
				outPath := c.Path("out")
				fmt.Printf("\nRendering templates from '%s' to '%s'...\n", inPath, outPath)
				return renderTemplates(c.Path("dir"), c.Path("out"), values)
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		exitOnError(err, "error")
	}
}

func generateShootTemplate(provider, outPath string) error {
	shootTemplate, err := templates.GenerateShootTemplate(provider)
	if err != nil {
		return fmt.Errorf("error when generating Shoot tamplate: %s", err.Error())
	}

	err = ioutil.WriteFile(outPath, shootTemplate, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error when writing template to file: %s", err.Error())
	}

	return nil
}

func renderTemplates(inDir, outPathDir string, values templates.Values) error {
	dir, err := ioutil.ReadDir(inDir)
	if err != nil {
		return fmt.Errorf("error while reading templates directory: %s", err.Error())
	}

	if err := ensureDirExists(outPathDir); err != nil {
		return fmt.Errorf("error when ensuring output directory exists: %s", err.Error())
	}

	for _, file := range dir {
		if file.IsDir() {
			continue
		}

		content, err := ioutil.ReadFile(path.Join(inDir, file.Name()))
		if err != nil {
			return fmt.Errorf("error while reading %s file: %s", file.Name(), err.Error())
		}

		rendered, err := templates.RenderTemplate(string(content), values)
		if err != nil {
			// If failed to render, the fail may not be template - log error and continue
			fmt.Printf("\nerror while rendering %s file: %s\n", file.Name(), err.Error())
			continue
		}

		err = ioutil.WriteFile(path.Join(outPathDir, file.Name()), rendered, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error when writing rendered template to file: %s", err.Error())
		}
	}

	return nil
}

func ensureDirExists(dirName string) error {
	err := os.MkdirAll(dirName, os.ModePerm)
	if err == nil || os.IsExist(err) {
		return nil
	} else {
		return err
	}
}

func exitOnError(err error, context string) {
	if err != nil {
		wrappedError := fmt.Errorf("%s: %s", context, err.Error())
		log.Fatal(wrappedError)
	}
}
