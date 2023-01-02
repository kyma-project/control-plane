package kubeconfig

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"

	"gopkg.in/yaml.v2"
)

type Config struct {
	AllowOrigins string
}

type Builder struct {
	provisionerClient provisioner.Client
}

func NewBuilder(provisionerClient provisioner.Client) *Builder {
	return &Builder{
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

func (b *Builder) BuildFromAdminKubeconfig(instance *internal.Instance, adminKubeconfig string) (string, error) {
	status, err := b.provisionerClient.RuntimeStatus(instance.GlobalAccountID, instance.RuntimeID)
	if err != nil {
		return "", fmt.Errorf("while fetching runtime status from provisioner: %w", err)
	}

	var kubeCfg kubeconfig
	if adminKubeconfig == "" {
		if status.RuntimeConfiguration.Kubeconfig == nil {
			return "", fmt.Errorf("kubeconfig is nil (nil response from Provisioner)")
		}
		adminKubeconfig = *status.RuntimeConfiguration.Kubeconfig
	}
	err = yaml.Unmarshal([]byte(adminKubeconfig), &kubeCfg)
	if err != nil {
		return "", fmt.Errorf("while unmarshaling kubeconfig: %w", err)
	}

	if err := b.validKubeconfig(kubeCfg); err != nil {
		return "", fmt.Errorf("while validation kubeconfig fetched by provisioner: %w", err)
	}

	return b.parseTemplate(kubeconfigData{
		ContextName:   kubeCfg.CurrentContext,
		CAData:        kubeCfg.Clusters[0].Cluster.CertificateAuthorityData,
		ServerURL:     kubeCfg.Clusters[0].Cluster.Server,
		OIDCIssuerURL: status.RuntimeConfiguration.ClusterConfig.OidcConfig.IssuerURL,
		OIDCClientID:  status.RuntimeConfiguration.ClusterConfig.OidcConfig.ClientID,
	})
}

func (b *Builder) Build(instance *internal.Instance) (string, error) {
	return b.BuildFromAdminKubeconfig(instance, "")
}

func (b *Builder) parseTemplate(payload kubeconfigData) (string, error) {
	var result bytes.Buffer
	t := template.New("kubeconfigParser")
	t, err := t.Parse(kubeconfigTemplate)
	if err != nil {
		return "", fmt.Errorf("while parsing kubeconfig template: %w", err)
	}

	err = t.Execute(&result, payload)
	if err != nil {
		return "", fmt.Errorf("while executing kubeconfig template: %w", err)
	}
	return result.String(), nil
}

func (b *Builder) validKubeconfig(kc kubeconfig) error {
	if kc.CurrentContext == "" {
		return fmt.Errorf("current context is empty")
	}
	if len(kc.Clusters) == 0 {
		return fmt.Errorf("there are no defined clusters")
	}
	if kc.Clusters[0].Cluster.CertificateAuthorityData == "" || kc.Clusters[0].Cluster.Server == "" {
		return fmt.Errorf("there are no cluster certificate or server info")
	}

	return nil
}
