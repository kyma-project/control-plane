package model

type KymaComponent string

type KymaProfile string

type KymaConfig struct {
	ID                  string
	Release             Release
	Profile             *KymaProfile
	Components          []KymaComponentConfig
	GlobalConfiguration Configuration
	ClusterID           string
	Active              bool
}

func (c KymaConfig) GetComponentConfig(name string) (KymaComponentConfig, bool) {
	for _, c := range c.Components {
		if string(c.Component) == name {
			return c, true
		}
	}

	return KymaComponentConfig{}, false
}

type Release struct {
	Id            string
	Version       string
	TillerYAML    string
	InstallerYAML string
}

type GithubRelease struct {
	Id         int     `json:"id"`
	Name       string  `json:"name"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
}

type Asset struct {
	Name string `json:"name"`
	Url  string `json:"browser_download_url"`
}

type KymaComponentConfig struct {
	ID             string
	KymaConfigID   string
	Component      KymaComponent
	Namespace      string
	SourceURL      *string
	ComponentOrder int
	Prerequisites  Prerequisites
	Configuration  Configuration
}

func (c KymaComponentConfig) HasPrerequisites() bool {
	return len(c.Prerequisites.Secrets) > 0 || len(c.Prerequisites.Certificates) > 0
}

type Prerequisites struct {
	Secrets      []SecretPrerequisite              `json:"secrets"`
	Certificates []GardenerCertificatePrerequisite `json:"certificates"`
}

type SecretPrerequisite struct {
	ResourceName string        `json:"resourceName"`
	Entries      []SecretEntry `json:"entries"`
}

type SecretEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func NewSecretEntry(key, val string) SecretEntry {
	return SecretEntry{
		Key:   key,
		Value: val,
	}
}

type GardenerCertificatePrerequisite struct {
	ResourceName string `json:"resourceName"`
	SecretName   string `json:"secretName"`
	CommonName   string `json:"commonName"`
}

func NewGardenerCertificatePrerequisite(resourceName, secretName, commonName string) GardenerCertificatePrerequisite {
	return GardenerCertificatePrerequisite{
		ResourceName: resourceName,
		SecretName:   secretName,
		CommonName:   commonName,
	}
}

type Configuration struct {
	ConfigEntries    []ConfigEntry `json:"configEntries"`
	ConflictStrategy string        `json:"conflictStrategy"`
}

type ConfigEntry struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

func NewConfigEntry(key, val string, secret bool) ConfigEntry {
	return ConfigEntry{
		Key:    key,
		Value:  val,
		Secret: secret,
	}
}

const (
	EvaluationProfile KymaProfile = "EVALUATION"
	ProductionProfile KymaProfile = "PRODUCTION"
)
