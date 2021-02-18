package auditlog

import (
	"errors"
	"github.com/Masterminds/semver"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/sirupsen/logrus"
	"text/template"
)

type Config struct {
	URL           string `envconfig:"APP_AUDITLOG_URL"`
	User          string `envconfig:"APP_AUDITLOG_USER"`
	Password      string `envconfig:"APP_AUDITLOG_PASSWORD"`
	Tenant        string `envconfig:"APP_AUDITLOG_TENANT"`
	EnableSeqHttp bool   `envconfig:"APP_AUDITLOG_ENABLE_SEQ_HTTP"`
}

type Overrides struct {
	Host string
	Port string
	Path string
	HttpPlugin string
	ClsOverrides *cls.ClsOverrides
	Config Config
}

func getExtraConf119(log logrus.FieldLogger)(*template.Template, error){
	//TODO: Chnage to ParseFile
	tmpl, err := template.New("test").Parse("[INPUT]\n        Name              tail\n        Tag               dex.*\n        Path              /var/log/containers/*_dex-*.log\n        DB                /var/log/flb_kube_dex.db\n        parser            docker\n        Mem_Buf_Limit     5MB\n        Skip_Long_Lines   On\n        Refresh_Interval  10\n[FILTER]\n        Name    lua\n        Match   dex.*\n        script  script.lua\n        call    reformat\n[FILTER]\n        Name    grep\n        Match   dex.*\n        Regex   time .*\n[FILTER]\n        Name    grep\n        Match   dex.*\n        Regex   data .*\\\"xsuaa\n[OUTPUT]\n        Name             {{.HttpPlugin}}\n        Match            dex.*\n        Retry_Limit      False\n        Host             {{.Host}}\n        Port             {{.Port}}\n        URI              {{.Path}}security-events\n        Header           Content-Type application/json\n        HTTP_User        {{.Config.User}}\n        HTTP_Passwd      {{.Config.Password}}\n        Format           json_stream\n        tls              on\n[OUTPUT]\n\t\tName              http\n\t\tMatch             *\n\t\tHost              {{.ClsOverrides.FluentdEndPoint}}\n\t\tPort              443\n\t\tHTTP_User         {{.ClsOverrides.FluentdUsername}}\n\t\tHTTP_Passwd       {{.ClsOverrides.FluentdPassword}}\n\t\ttls               true\n\t\ttls.verify        true\n\t\ttls.debug         1\n\t\tURI               /\n\t\tFormat            json")
	if err != nil {
		log.Errorf("Template error: %v", err)
		return nil, err
	}
	return tmpl, err
}

func getExtraConf120(log logrus.FieldLogger)(*template.Template, error){
	//TODO: Chnage to ParseFile
	tmpl, err := template.New("test").Parse("[INPUT]\n        Name              tail\n        Tag               dex.*\n        Path              /var/log/containers/*_dex-*.log\n        DB                /var/log/flb_kube_dex.db\n        parser            docker\n        Mem_Buf_Limit     5MB\n        Skip_Long_Lines   On\n        Refresh_Interval  10\n[FILTER]\n        Name    lua\n        Match   dex.*\n        script  script.lua\n        call    reformat\n[FILTER]\n        Name    grep\n        Match   dex.*\n        Regex   time .*\n[FILTER]\n        Name    grep\n        Match   dex.*\n        Regex   data .*\\\"xsuaa\n[OUTPUT]\n        Name             {{.HttpPlugin}}\n        Match            dex.*\n        Retry_Limit      False\n        Host             {{.Host}}\n        Port             {{.Port}}\n        URI              {{.Path}}security-events\n        Header           Content-Type application/json\n        HTTP_User        {{.Config.User}}\n        HTTP_Passwd      {{.Config.Password}}\n        Format           json_stream\n        tls              on")
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