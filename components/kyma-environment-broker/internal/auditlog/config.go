package auditlog

import (
	"errors"
	"github.com/gobuffalo/packr"
	"text/template"

	"github.com/Masterminds/semver"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/sirupsen/logrus"
)
//go:generate packr -v

type Config struct {
	URL           string `envconfig:"APP_AUDITLOG_URL"`
	User          string `envconfig:"APP_AUDITLOG_USER"`
	Password      string `envconfig:"APP_AUDITLOG_PASSWORD"`
	Tenant        string `envconfig:"APP_AUDITLOG_TENANT"`
	EnableSeqHttp bool   `envconfig:"APP_AUDITLOG_ENABLE_SEQ_HTTP"`
}

type Overrides struct {
	Host         string
	Port         string
	Path         string
	HttpPlugin   string
	ClsOverrides *cls.ClsOverrides
	Config       Config
}

func getExtraConf119(log logrus.FieldLogger) (*template.Template, error) {
	box := packr.NewBox("./templates")
	yamlFile, err := box.FindString("extra_conf_kyma_119.yaml")
	tmpl, err := template.New("extra_conf").Parse(yamlFile)
	if err != nil {
		log.Errorf("Template error: %v", err)
		return nil, err
	}
	return tmpl, err
}

func getExtraConf120(log logrus.FieldLogger) (*template.Template, error) {
	box := packr.NewBox("./templates")
	yamlFile, err := box.FindString("extra_conf_kyma.yaml")
	tmpl, err := template.New("extra_conf").Parse(yamlFile)
	if err != nil {
		log.Errorf("Template error: %v", err)
		return nil, err
	}
	return tmpl, err
}

func GetExtraConf(KymaVersion string, log logrus.FieldLogger) (*template.Template, error) {
	c, err := semver.NewConstraint("< 1.20.x")
	if err != nil {
		return nil, errors.New("unable to parse constraint for kyma version to set correct fluent bit plugin")
	}
	v, err := semver.NewVersion(KymaVersion)
	check := c.Check(v)
	if check {
		return getExtraConf119(log)
	} else {
		return getExtraConf120(log)
	}

}
