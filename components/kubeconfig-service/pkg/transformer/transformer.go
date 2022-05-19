package transformer

import (
	"bytes"
	"html/template"

	"github.com/kyma-project/control-plane/components/kubeconfig-service/pkg/env"

	"gopkg.in/yaml.v2"
)

//Client Wrapper for transformer operations
type Client struct {
	ContextName   string
	CAData        string
	ServerURL     string
	OIDCIssuerURL string
	OIDCClientID  string
	SaToken       string
	UserID        string
}

//NewClient Create new instance of TransformerClient
func NewClient(rawKubeCfg string, userID string) (*Client, error) {
	var kubeCfg kubeconfig
	err := yaml.Unmarshal([]byte(rawKubeCfg), &kubeCfg)
	if err != nil {
		return nil, err
	}
	return &Client{
		ContextName:   kubeCfg.CurrentContext,
		CAData:        kubeCfg.Clusters[0].Cluster.CertificateAuthorityData,
		ServerURL:     kubeCfg.Clusters[0].Cluster.Server,
		OIDCClientID:  env.Config.OIDC.Kubeconfig.ClientID,
		OIDCIssuerURL: env.Config.OIDC.Kubeconfig.IssuerURL,
		SaToken:       "",
		UserID:        userID,
	}, nil
}

//TransformKubeconfig injects OIDC data into raw kubeconfig structure
func (c *Client) TransformKubeconfig(template string) ([]byte, error) {
	out, err := c.parseTemplate(template)
	if err != nil {
		return nil, err
	}

	return []byte(out), nil
}

func (c *Client) parseTemplate(templ string) (string, error) {
	var result bytes.Buffer
	t := template.New("kubeconfigParser")
	t, err := t.Parse(templ)
	if err != nil {
		return "", err
	}

	err = t.Execute(&result, c)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}