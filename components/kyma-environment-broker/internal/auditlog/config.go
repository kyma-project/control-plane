package auditlog

type Config struct {
	URL           string `envconfig:"APP_AUDITLOG_URL"`
	User          string `envconfig:"APP_AUDITLOG_USER"`
	Password      string `envconfig:"APP_AUDITLOG_PASSWORD"`
	Tenant        string `envconfig:"APP_AUDITLOG_TENANT"`
	EnableSeqHttp bool   `envconfig:"APP_AUDITLOG_ENABLE_SEQ_HTTP"`
	Disabled                    bool `envconfig:"default=false"`
}
