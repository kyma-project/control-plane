package transformer_test

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kubeconfig-service/pkg/env"
	"github.com/kyma-project/control-plane/components/kubeconfig-service/pkg/transformer"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSpec(t *testing.T) {

	env.Config.OIDC.Kubeconfig.ClientID = testClientID
	env.Config.OIDC.Kubeconfig.IssuerURL = testIssuerURL

	// Only pass t into top-level Convey calls
	Convey("NewClient()", t, func() {
		Convey("when given correct raw KubeConfig", func() {
			Convey("Should return a Client", func() {
				//given, when
				c, err := transformer.NewClient(testInputRawKubeconfig)
				//then
				So(err, ShouldBeNil)
				So(c.ContextName, ShouldEqual, "test--aa1234b")
				So(c.CAData, ShouldEqual, "LS0FakeFakeQo=")
				So(c.ServerURL, ShouldEqual, "https://api.kymatest.com")
				So(c.OIDCClientID, ShouldEqual, testClientID)
				So(c.OIDCIssuerURL, ShouldEqual, testIssuerURL)
			})
		})
	})

	Convey("client.TransformKubeconfig()", t, func() {
		Convey("Should return transformed kubeconfig", func() {
			//given
			c, err := transformer.NewClient(testInputRawKubeconfig)
			So(err, ShouldBeNil)
			//when
			res, err := c.TransformKubeconfig(transformer.KubeconfigTemplate)
			//then
			So(err, ShouldBeNil)
			So(string(res), ShouldEqual, expectedTransformedKubeconfig)
		})
	})
}

const (
	testClientID     = "testClientId"
	testClientSecret = "testClientSecret"
	testIssuerURL    = "testIssuerURL"

	testInputRawKubeconfig = `
apiVersion: v1
kind: Config
clusters:
  - name: test--aa1234b
    cluster:
      server: 'https://api.kymatest.com'
      certificate-authority-data: LS0FakeFakeQo=
contexts:
  - name: test--aa1234b
    context:
      cluster: test--aa1234b
      user: test--aa1234b-token
current-context: test--aa1234b
users:
  - name: test--aa1234b-token
    user:
      token: 7WFakeFakeK
`

	expectedTransformedKubeconfig = `
---
apiVersion: v1
kind: Config
current-context: test--aa1234b
clusters:
- name: test--aa1234b
  cluster:
    certificate-authority-data: LS0FakeFakeQo=
    server: https://api.kymatest.com
contexts:
- name: test--aa1234b
  context:
    cluster: test--aa1234b
    user: test--aa1234b
users:
- name: test--aa1234b
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - "--oidc-issuer-url=testIssuerURL"
      - "--oidc-client-id=testClientId"
      - "--oidc-extra-scope=email"
      - "--oidc-extra-scope=openid"
      command: kubectl-oidc_login
      installHint: |
        kubelogin plugin is required to proceed with authentication
        # Homebrew (macOS and Linux)
        brew install int128/kubelogin/kubelogin

        # Krew (macOS, Linux, Windows and ARM)
        kubectl krew install oidc-login

        # Chocolatey (Windows)
        choco install kubelogin
`
)
