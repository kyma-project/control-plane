package auditlog

import (
	"errors"
	"fmt"
	"text/template"

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
	checkKymaVersion, err := cls.IsKymaVersionAtLeast_1_20(KymaVersion)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("unable to check kyma version: %v", err))
	}

	if !checkKymaVersion {
		return template.New("fluent-bit-extra-conf-for-kyma-1-19").Parse(templates.FluentBitExtraConfForKyma1_19)
	} else {
		return template.New("fluent-bit-extra-conf-for-kyma-above-1-19").Parse(templates.FluentBitExtraConf)
	}
}
