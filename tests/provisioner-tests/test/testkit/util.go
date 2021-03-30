package testkit

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	gqlschema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
)

const (
	AWS   = "AWS"
	Azure = "Azure"
	GCP   = "GCP"
)

func WaitForFunction(interval, timeout time.Duration, isDone func() bool) error {
	done := time.After(timeout)

	for {
		if isDone() {
			return nil
		}

		select {
		case <-done:
			return errors.New("timeout waiting for condition")
		default:
			time.Sleep(interval)
		}
	}
}

func IsTillerPresent(httpClient http.Client, kymaVersion string) (bool, error) {
	tillerYAMLURL := fmt.Sprintf("https://storage.googleapis.com/kyma-prow-artifacts/%s/tiller.yaml", kymaVersion)

	resp, err := httpClient.Get(tillerYAMLURL)
	if err != nil {
		return false, errors.Wrapf(err, "while executing get request on url: %q", tillerYAMLURL)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, errors.Errorf("received unexpected http status %d", resp.StatusCode)
	}

	reqBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, errors.Wrap(err, "while reading body")
	}

	return string(reqBody) != "", nil
}

func GetAndParseInstallerCR(installationCRURL string) ([]*gqlschema.ComponentConfigurationInput, error) {
	resp, err := http.Get(installationCRURL)
	if err != nil {
		return nil, fmt.Errorf("Error fetching installation CR: %s", err.Error())
	}
	crContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading body of installation CR GET response: %s", err.Error())
	}

	installationCR := v1alpha1.Installation{}
	err = yaml.NewDecoder(bytes.NewBuffer(crContent)).Decode(&installationCR)
	if err != nil {
		return nil, fmt.Errorf("Error decoding installer CR: %s", err.Error())
	}
	var components = make([]*gqlschema.ComponentConfigurationInput, 0, len(installationCR.Spec.Components))
	for _, component := range installationCR.Spec.Components {
		in := &gqlschema.ComponentConfigurationInput{
			Component: component.Name,
			Namespace: component.Namespace,
		}
		components = append(components, in)
	}
	return components, nil
}

func createInstallationCRURL(kymaVersion string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/kyma-project/kyma/%s/installation/resources/installer-cr-cluster-runtime.yaml.tpl", kymaVersion)
}

func closeBody(closer io.ReadCloser) {
	if err := closer.Close(); err != nil {
		logrus.Warnf("failed to close read closer: %v", err)
	}
}

func toLowerCase(provider string) string {
	return strings.ToLower(provider)
}

func intToPtr(i int) *int {
	return &i
}

func strToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func boolToPtr(b bool) *bool {
	return &b
}
