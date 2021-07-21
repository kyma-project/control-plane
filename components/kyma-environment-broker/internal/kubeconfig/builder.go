package kubeconfig

import (
	"bytes"
	"text/template"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	IssuerURL    string
	ClientID     string
	AllowOrigins string
}

type Builder struct {
	config            Config
	provisionerClient provisioner.Client
}

func NewBuilder(cfg Config, provisionerClient provisioner.Client) *Builder {
	return &Builder{
		config:            cfg,
		provisionerClient: provisionerClient,
	}
}

type kubeconfigData struct {
	ContextName   string
	CAData        string
	ServerURL     string
	OIDCIssuerURL string
	OIDCClientID  string
}

func (b *Builder) Build(instance *internal.Instance) (string, error) {
	status, err := b.provisionerClient.RuntimeStatus(instance.GlobalAccountID, instance.RuntimeID)
	if err != nil {
		return "", errors.Wrapf(err, "while fetching runtime status from provisioner")
	}

	var kubeCfg kubeconfig
	err = yaml.Unmarshal([]byte(*status.RuntimeConfiguration.Kubeconfig), &kubeCfg)
	if err != nil {
		return "", errors.Wrapf(err, "while unmarshaling kubeconfig")
	}

	if err := b.validKubeconfig(kubeCfg); err != nil {
		return "", errors.Wrap(err, "while validation kubeconfig fetched by provisioner")
	}

	return b.parseTemplate(kubeconfigData{
		ContextName:   kubeCfg.CurrentContext,
		CAData:        kubeCfg.Clusters[0].Cluster.CertificateAuthorityData,
		ServerURL:     kubeCfg.Clusters[0].Cluster.Server,
		OIDCIssuerURL: b.config.IssuerURL,
		OIDCClientID:  b.config.ClientID,
	})
}

func (b *Builder) parseTemplate(payload kubeconfigData) (string, error) {
	var result bytes.Buffer
	t := template.New("kubeconfigParser")
	t, err := t.Parse(kubeconfigTemplate)
	if err != nil {
		return "", errors.Wrap(err, "while parsing kubeconfig template")
	}

	err = t.Execute(&result, payload)
	if err != nil {
		return "", errors.Wrap(err, "while executing kubeconfig template")
	}
	return result.String(), nil
}

func (b *Builder) validKubeconfig(kc kubeconfig) error {
	if kc.CurrentContext == "" {
		return errors.New("current context is empty")
	}
	if len(kc.Clusters) == 0 {
		return errors.New("there are no defined clusters")
	}
	if kc.Clusters[0].Cluster.CertificateAuthorityData == "" || kc.Clusters[0].Cluster.Server == "" {
		return errors.New("there are no cluster certificate or server info")
	}

	return nil
}
