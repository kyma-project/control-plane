package auditlog

import (
	"errors"
	"text/template"

	"github.com/gobuffalo/packr"

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
	ClsOverrides *cls.ClsOverrideParams
	Config       Config
}

func getExtraConfForKyma1_19(log logrus.FieldLogger) (*template.Template, error) {
	box := packr.NewBox("./templates")
	confFile, err := box.FindString("extra_conf_kyma_119.conf")
	tmpl, err := template.New("extra_conf").Parse(confFile)
	if err != nil {
		log.Errorf("Template error: %v", err)
		return nil, err
	}
	return tmpl, err
}

func getExtraConfForKyma1_20(log logrus.FieldLogger) (*template.Template, error) {
	box := packr.NewBox("./templates")
	confFile, err := box.FindString("extra_conf_kyma.conf")
	tmpl, err := template.New("extra_conf").Parse(confFile)
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
		return getExtraConfForKyma1_19(log)
	} else {
		return getExtraConfForKyma1_20(log)
	}

}
