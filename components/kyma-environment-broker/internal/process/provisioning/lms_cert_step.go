package provisioning

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/lms"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"crypto/x509/pkix"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	pollingInterval          = 15 * time.Second
	certPollingTimeout       = 30 * time.Minute
	tenantReadyRetryInterval = 30 * time.Second
	lmsTimeout               = 30 * time.Minute
	kibanaURLLabelKey        = "operator_lmsUrl"
)

type LmsClient interface {
	RequestCertificate(tenantID string, subject pkix.Name) (id string, privateKey []byte, err error)
	GetCertificateByURL(url string) (cert string, found bool, err error)
	GetCACertificate(tenantID string) (cert string, found bool, err error)
	GetTenantStatus(tenantID string) (status lms.TenantStatus, err error)
	GetTenantInfo(tenantID string) (status lms.TenantInfo, err error)
}

type lmsCertStep struct {
	LmsStep
	provider            LmsClient
	normalizationRegexp *regexp.Regexp
}

func NewLmsCertificatesStep(certProvider LmsClient, os storage.Operations, isMandatory bool) *lmsCertStep {
	return &lmsCertStep{
		LmsStep: LmsStep{
			operationManager: process.NewProvisionOperationManager(os),
			isMandatory:      isMandatory,
			expirationTime:   lmsTimeout,
		},
		provider:            certProvider,
		normalizationRegexp: regexp.MustCompile("[^a-zA-Z0-9]+"),
	}
}

func (s *lmsCertStep) Name() string {
	return "Request_LMS_Certificates"
}

// Run executes getting LMS certificates steps, which means:
// 1. check if the tenant is ready
// 2. request certificates
// 3. poll CA and signed certificates
func (s *lmsCertStep) Run(operation internal.ProvisioningOperation, l logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.Lms.Failed {
		l.Info("LMS has failed, skipping")
		return operation, 0, nil
	}
	logger := l.WithField("LMSTenant", operation.Lms.TenantID)

	if operation.Lms.TenantID == "" {
		logger.Error("Create LMS Tenant step must be run before")
		return operation, 0, errors.New("the step needs to be run after 'Create LMS tenant' step")
	}

	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		logger.Errorf("Unable to get provisioning parameters: %s", err.Error())
		return operation, 0, errors.New("unable to get provisioning parameters")
	}

	// check if LMS tenant is ready
	status, err := s.provider.GetTenantStatus(operation.Lms.TenantID)
	if err != nil {
		return s.handleError(
			operation,
			logger,
			time.Since(operation.Lms.RequestedAt),
			"Unable to get LMS Tenant status",
			err)
	}
	if !(status.ElasticsearchDNSResolves && status.KibanaDNSResolves) {
		logger.Infof("LMS tenant not ready: elasticDNS=%v, kibanaDNS=%v", status.ElasticsearchDNSResolves, status.KibanaDNSResolves)
		if time.Since(operation.Lms.RequestedAt) > lmsTimeout {
			logger.Error("Setting LMS operation failed - tenant provisioning timed out")
			return s.failLmsAndUpdate(operation, "LMS Tenant provisioning timeout")
		}
		return operation, tenantReadyRetryInterval, nil
	}

	tenantInfo, err := s.provider.GetTenantInfo(operation.Lms.TenantID)
	if err != nil {
		return s.handleError(
			operation,
			logger,
			time.Since(operation.Lms.RequestedAt),
			"Unable to get LMS Tenant info",
			err)
	}

	// request certificates
	subj := pkix.Name{
		CommonName:         "fluentbit", // do not modify
		Organization:       []string{pp.ErsContext.GlobalAccountID},
		OrganizationalUnit: []string{uuid.New().String()},
	}
	certURL, pKey, err := s.provider.RequestCertificate(operation.Lms.TenantID, subj)
	if err != nil {
		logger.Errorf("Unable to request LMS Certificates %s", err.Error())
		return operation, 5 * time.Second, nil
	}
	logger.Infof("Signed Certificate URL: %s", certURL)

	var signedCert string
	var caCert string

	// certs cannot be stored so there is a need to poll until certs are ready
	// get Signed Certificate
	err = wait.PollImmediate(pollingInterval, certPollingTimeout, func() (done bool, err error) {
		c, found, err := s.provider.GetCertificateByURL(certURL)
		if err != nil {
			logger.Warnf("Unable to get LMS Signed Certificate: %s, retrying", err.Error())
			return false, nil
		}
		if !found {
			logger.Info("LMS Signed Certificate not ready")
			return false, nil
		}
		signedCert = c
		return true, nil
	})
	if err != nil {
		logger.Errorf("Setting LMS operation failed: %s", err.Error())
		return s.failLmsAndUpdate(operation, "Getting LMS Signed Certificate timeout")
	}

	// get CA cert
	err = wait.PollImmediate(pollingInterval, certPollingTimeout, func() (done bool, err error) {
		c, found, err := s.provider.GetCACertificate(operation.Lms.TenantID)
		if err != nil {
			logger.Warnf("Unable to get LMS CA Certificate: %s", err.Error())
			return false, nil
		}
		if !found {
			logger.Info("LMS Ca Certificate not ready")
			return false, nil
		}
		caCert = c
		return true, nil
	})
	if err != nil {
		logger.Errorf("Setting LMS operation failed: %s", err.Error())
		return s.failLmsAndUpdate(operation, "getting LMS CA certificate timeout")
	}

	operation.InputCreator.SetLabel(kibanaURLLabelKey, fmt.Sprintf("https://kibana.%s", tenantInfo.DNS))

	operation.InputCreator.AppendOverrides("logging", []*gqlschema.ConfigEntryInput{
		{Key: "fluent-bit.conf.Output.forward.enabled", Value: "true"},
		{Key: "fluent-bit.conf.Output.forward.Match", Value: "kube.*"},

		{Key: "fluent-bit.backend.forward.host", Value: fmt.Sprintf("forward.%s", tenantInfo.DNS)},
		{Key: "fluent-bit.backend.forward.port", Value: "8443"},
		{Key: "fluent-bit.backend.forward.tls.enabled", Value: "true"},
		{Key: "fluent-bit.backend.forward.tls.verify", Value: "On"},

		// certs and private key must be encoded by base64
		{Key: "fluent-bit.backend.forward.tls.ca", Value: base64.StdEncoding.EncodeToString([]byte(caCert))},
		{Key: "fluent-bit.backend.forward.tls.cert", Value: base64.StdEncoding.EncodeToString([]byte(signedCert))},
		{Key: "fluent-bit.backend.forward.tls.key", Value: base64.StdEncoding.EncodeToString(pKey)},

		// record modifier filter
		{Key: "fluent-bit.conf.Filter.record_modifier.enabled", Value: "true"},
		{Key: "fluent-bit.conf.Filter.record_modifier.Match", Value: "kube.*"},
		{Key: "fluent-bit.conf.Filter.record_modifier.Key", Value: "subaccount_id"},
		{Key: "fluent-bit.conf.Filter.record_modifier.Value", Value: pp.ErsContext.SubAccountID}, // cluster_name is a tag added to log entry, allows to filter logs by a cluster
		//kubernetes filter should not parse the document to avoid indexing on LMS side
		{Key: "fluent-bit.conf.Filter.Kubernetes.Merge_Log", Value: "Off"},
		//input should not contain dex logs as it contains sensitive data
		{Key: "fluent-bit.conf.Input.Kubernetes.Exclude_Path", Value: "/var/log/containers/*_dex-*.log,/var/log/containers/*_kcproxy-*.log"},
	})
	return operation, 0, nil
}

type LmsStep struct {
	operationManager *process.ProvisionOperationManager
	isMandatory      bool
	expirationTime   time.Duration
}

func (s *LmsStep) handleError(operation internal.ProvisioningOperation, log logrus.FieldLogger, since time.Duration, msg string, err error) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 10*time.Second, time.Minute*30, log)
	default:
		if since < s.expirationTime {
			return operation, tenantReadyRetryInterval, nil
		}
		return s.failLmsAndUpdate(operation, "getting LMS tenant failed")
	}
}

func (s *LmsStep) failLmsAndUpdate(operation internal.ProvisioningOperation, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	operation.Lms.Failed = true
	if s.isMandatory {
		return s.operationManager.OperationFailed(operation, msg)
	}
	modifiedOp, retry := s.operationManager.UpdateOperation(operation)
	return modifiedOp, retry, nil
}
