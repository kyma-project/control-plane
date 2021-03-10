package auditlog

import (
	"errors"
	"net"
	"net/url"
	"text/template"

	pkgErrors "github.com/pkg/errors"

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

type OverrideParams struct {
	Host              string
	Port              string
	Path              string
	HttpPlugin        string
	ClsOverrideParams *cls.OverrideParams
	Config            Config
}

func PrepareOverrideParams(config *Config, secretKey string, encrptedClsOverrides string) (*OverrideParams, error) {
	u, err := url.Parse(config.URL)
	if err != nil {
		return nil, pkgErrors.Wrapf(err, "while parsing the Audit Log URL")
	}

	if u.Path == "" {
		return nil, errors.New("There is no Path passed in the URL")
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, pkgErrors.Wrapf(err, "while splitting Audit Log URL")
	}
	if port == "" {
		port = "443"
	}

	fluentBitPluginName := "http"
	if config.EnableSeqHttp {
		fluentBitPluginName = "sequentialhttp"
	}

	decryptedOverrideParams, err := cls.DecryptOverrides(secretKey, encrptedClsOverrides)
	if err != nil {
		return nil, pkgErrors.Wrapf(err, "while decrypting cls overrides")
	}

	return &OverrideParams{
		Host:              host,
		Port:              port,
		Path:              u.Path,
		HttpPlugin:        fluentBitPluginName,
		ClsOverrideParams: decryptedOverrideParams,
		Config:            *config,
	}, nil
}

func GetExtraConfTemplate(KymaVersion string) (*template.Template, error) {
	checkKymaVersion, err := cls.IsKymaVersionAtLeast_1_20(KymaVersion)
	if err != nil {
		return nil, pkgErrors.Wrapf(err, "while checking Kyma version")
	}

	if !checkKymaVersion {
		return template.New("fluent-bit-extra-conf-for-kyma-1-19").Parse(templates.FluentBitExtraConfForKyma1_19)
	}

	return template.New("fluent-bit-extra-conf-for-kyma-above-1-19").Parse(templates.FluentBitExtraConf)
}
