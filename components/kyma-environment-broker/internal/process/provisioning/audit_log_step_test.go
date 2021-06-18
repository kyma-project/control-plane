package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/auditlog"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLog_ScriptFileDoesNotExist(t *testing.T) {
	// given
	mm := afero.NewMemMapFs()

	repo := storage.NewMemoryStorage().Operations()
	cfg := auditlog.Config{
		URL:      "host1",
		User:     "aaaa",
		Password: "aaaa",
		Tenant:   "tenant",
	}
	svc := NewAuditLogOverridesStep(mm, repo, cfg)

	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			ProvisioningParameters: internal.ProvisioningParameters{ErsContext: internal.ERSContext{SubAccountID: "1234567890"}},
		},
	}
	err := repo.InsertProvisioningOperation(operation)
	require.NoError(t, err)

	// when
	_, _, err = svc.Run(operation, NewLogDummy())
	//then
	require.Error(t, err)
	require.EqualError(t, err, "open /auditlog-script/script: file does not exist")

}

func TestAuditLog_HappyPath(t *testing.T) {
	// given
	mm := afero.NewMemMapFs()

	fileScript := `
func myScript() {
foo: sub_account_id
bar: tenant_id
return "fooBar"
}
`

	err := afero.WriteFile(mm, "/auditlog-script/script", []byte(fileScript), 0755)
	if err != nil {
		t.Fatalf("Unable to write contents to file: audit-log-script!!: %v", err)
	}

	repo := storage.NewMemoryStorage().Operations()
	cfg := auditlog.Config{
		URL:      "https://host1:8080/aaa/v2/",
		User:     "aaaa",
		Password: "aaaa",
		Tenant:   "tenant",
	}
	svc := NewAuditLogOverridesStep(mm, repo, cfg)

	inputCreatorMock := &automock.ProvisionerInputCreator{}
	defer inputCreatorMock.AssertExpectations(t)

	expectedOverride_config := `
[INPUT]
    Name              tail
    Tag               dex.*
    Path              /var/log/containers/*_dex-*.log
    DB                /var/log/flb_kube_dex.db
    parser            docker
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Refresh_Interval  10
[FILTER]
    Name    lua
    Match   dex.*
    script  script.lua
    call    reformat
[FILTER]
    Name    grep
    Match   dex.*
    Regex   time .*
[FILTER]
    Name    grep
    Match   dex.*
    Regex   data .*\"xsuaa
[OUTPUT]
    Name             http
    Match            dex.*
    Retry_Limit      False
    Host             host1
    Port             8080
    URI              /aaa/v2/security-events
    Header           Content-Type application/json
    HTTP_User        ${AUDITLOG_USER}
    HTTP_Passwd      ${AUDITLOG_PASSWD}
    Format           json_stream
    tls              on
`
	expectedFileScript := `
func myScript() {
foo: 1234567890
bar: tenant
return "fooBar"
}
`

	expectedPorts := `- number: 8080
  name: https
  protocol: TLS`
	inputCreatorMock.On("AppendOverrides", "logging", []*gqlschema.ConfigEntryInput{
		{
			Key:   "fluent-bit.config.script",
			Value: expectedFileScript,
		},
		{
			Key:   "fluent-bit.config.extra",
			Value: expectedOverride_config,
		},
		{
			Key:    "fluent-bit.config.secrets.AUDITLOG_USER",
			Value:  "aaaa",
			Secret: ptr.Bool(true),
		},
		{
			Key:    "fluent-bit.config.secrets.AUDITLOG_PASSWD",
			Value:  "aaaa",
			Secret: ptr.Bool(true),
		},
		{
			Key:   "fluent-bit.externalServiceEntry.resolution",
			Value: "DNS",
		},
		{
			Key:   "fluent-bit.externalServiceEntry.hosts",
			Value: "- host1",
		},
		{
			Key:   "fluent-bit.externalServiceEntry.ports",
			Value: expectedPorts,
		},
	}).Return(nil).Once()

	operation := internal.ProvisioningOperation{
		InputCreator: inputCreatorMock,
		Operation: internal.Operation{

			ProvisioningParameters: internal.ProvisioningParameters{ErsContext: internal.ERSContext{SubAccountID: "1234567890"}},
		},
	}
	repo.InsertProvisioningOperation(operation)
	// when
	_, repeat, err := svc.Run(operation, NewLogDummy())
	//then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
}

func TestAuditLog_HappyPath_SeqHttp(t *testing.T) {
	// given
	mm := afero.NewMemMapFs()

	fileScript := `
func myScript() {
foo: sub_account_id
bar: tenant_id
return "fooBar"
}
`

	err := afero.WriteFile(mm, "/auditlog-script/script", []byte(fileScript), 0755)
	if err != nil {
		t.Fatalf("Unable to write contents to file: audit-log-script!!: %v", err)
	}

	repo := storage.NewMemoryStorage().Operations()
	cfg := auditlog.Config{
		URL:           "https://host1:8080/aaa/v2/",
		User:          "aaaa",
		Password:      "aaaa",
		Tenant:        "tenant",
		EnableSeqHttp: true,
	}
	svc := NewAuditLogOverridesStep(mm, repo, cfg)
	inputCreatorMock := &automock.ProvisionerInputCreator{}
	defer inputCreatorMock.AssertExpectations(t)
	expectedOverride_config := `
[INPUT]
    Name              tail
    Tag               dex.*
    Path              /var/log/containers/*_dex-*.log
    DB                /var/log/flb_kube_dex.db
    parser            docker
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Refresh_Interval  10
[FILTER]
    Name    lua
    Match   dex.*
    script  script.lua
    call    reformat
[FILTER]
    Name    grep
    Match   dex.*
    Regex   time .*
[FILTER]
    Name    grep
    Match   dex.*
    Regex   data .*\"xsuaa
[OUTPUT]
    Name             sequentialhttp
    Match            dex.*
    Retry_Limit      False
    Host             host1
    Port             8080
    URI              /aaa/v2/security-events
    Header           Content-Type application/json
    HTTP_User        ${AUDITLOG_USER}
    HTTP_Passwd      ${AUDITLOG_PASSWD}
    Format           json_stream
    tls              on
`
	expectedFileScript := `
func myScript() {
foo: 1234567890
bar: tenant
return "fooBar"
}
`

	expectedPorts := `- number: 8080
  name: https
  protocol: TLS`
	inputCreatorMock.On("AppendOverrides", "logging", []*gqlschema.ConfigEntryInput{
		{
			Key:   "fluent-bit.config.script",
			Value: expectedFileScript,
		},
		{
			Key:   "fluent-bit.config.extra",
			Value: expectedOverride_config,
		},
		{
			Key:    "fluent-bit.config.secrets.AUDITLOG_USER",
			Value:  "aaaa",
			Secret: ptr.Bool(true),
		},
		{
			Key:    "fluent-bit.config.secrets.AUDITLOG_PASSWD",
			Value:  "aaaa",
			Secret: ptr.Bool(true),
		},
		{
			Key:   "fluent-bit.externalServiceEntry.resolution",
			Value: "DNS",
		},
		{
			Key:   "fluent-bit.externalServiceEntry.hosts",
			Value: "- host1",
		},
		{
			Key:   "fluent-bit.externalServiceEntry.ports",
			Value: expectedPorts,
		},
	}).Return(nil).Once()

	operation := internal.ProvisioningOperation{
		RuntimeVersion: internal.RuntimeVersionData{
			Version: "1.20",
			Origin:  "foo",
		},
		InputCreator: inputCreatorMock,
		Operation: internal.Operation{
			ProvisioningParameters: internal.ProvisioningParameters{ErsContext: internal.ERSContext{SubAccountID: "1234567890"}},
		},
	}
	repo.InsertProvisioningOperation(operation)
	// when
	_, repeat, err := svc.Run(operation, NewLogDummy())
	//then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
}
