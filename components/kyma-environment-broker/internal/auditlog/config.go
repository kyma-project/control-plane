package auditlog

import (
	"errors"
	"fmt"
	"text/template"

	"github.com/Masterminds/semver"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/auditlog/templates"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
)

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

func GetExtraConfTemplate(KymaVersion string) (*template.Template, error) {
	c, err := semver.NewConstraint("< 1.20.x")
	if err != nil {
		return nil, errors.New("unable to parse constraint for kyma version %s to set correct fluent bit plugin")
	}

	version, err := semver.NewVersion(KymaVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to parse kyma version %s to set correct fluent bit plugin", KymaVersion)
	}

	check := c.Check(version)
	if check {
		return template.New("fluent-bit-extra-conf-for-kyma-1-19").Parse(templates.FluentBitExtraConfForKyma1_19)
	} else {
		return template.New("fluent-bit-extra-conf-for-kyma-above-1-19").Parse(templates.FluentBitExtraConf)
	}
}
