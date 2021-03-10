package upgrade_kyma

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/auditlog"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClsUpgradeAuditLogStep_ScriptFileDoesNotExist(t *testing.T) {
	// given
	mm := afero.NewMemMapFs()

	repo := storage.NewMemoryStorage().Operations()
	cfg := auditlog.Config{
		URL:      "host1",
		User:     "aaaa",
		Password: "aaaa",
		Tenant:   "tenant",
	}
	svc := NewClsUpgradeAuditLogOverridesStep(repo, cfg, "1234567890123456")
	svc.fs = mm

	operation := internal.UpgradeKymaOperation{
		Operation: internal.Operation{
			ProvisioningParameters: internal.ProvisioningParameters{ErsContext: internal.ERSContext{SubAccountID: "1234567890"}},
		},
	}
	err := repo.InsertUpgradeKymaOperation(operation)
	require.NoError(t, err)

	// when
	_, _, err = svc.Run(operation, logger.NewLogDummy())
	//then
	require.Error(t, err)
	require.EqualError(t, err, "Unable to read audit config script")

}

func TestClsUpgradeAuditLogStep_HappyPath(t *testing.T) {
	// given
	mm := afero.NewMemMapFs()

	fileScript := `
func myScript() {
foo: sub_account_id
bar: tenant_id
return "fooBar"
}
`
	overridesIn := cls.OverrideParams{
		FluentdEndPoint: "foo.bar",
		FluentdPassword: "fooPass",
		FluentdUsername: "fooUser",
		KibanaUrl:       "Kiib.url",
	}
	secretKey := "1234567890123456"
	encrypted, err := cls.EncryptOverrides(secretKey, &overridesIn)
	assert.NoError(t, err)

	err = afero.WriteFile(mm, "/auditlog-script/script", []byte(fileScript), 0755)
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
	svc := NewClsUpgradeAuditLogOverridesStep(repo, cfg, secretKey)
	svc.fs = mm

	inputCreatorMock := &automock.ProvisionerInputCreator{}
	defer inputCreatorMock.AssertExpectations(t)
	expectedOverride_conf := `
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
    HTTP_User        aaaa
    HTTP_Passwd      aaaa
    Format           json_stream
    tls              on
[OUTPUT]
    Name              http
    Match             *
    Host              foo.bar
    Port              443
    HTTP_User         fooUser
    HTTP_Passwd       fooPass
    tls               true
    tls.verify        true
    tls.debug         1
    URI               /
    Format            json`
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
    HTTP_User        aaaa
    HTTP_Passwd      aaaa
    Format           json_stream
    tls              on
[OUTPUT]
    Name              http
    Match             *
    Host              foo.bar
    Port              443
    HTTP_User         fooUser
    HTTP_Passwd       fooPass
    tls               true
    tls.verify        true
    tls.debug         1
    URI               /
    Format            json`
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
			Key:   "fluent-bit.conf.script",
			Value: expectedFileScript,
		},
		{
			Key:   "fluent-bit.conf.extra",
			Value: expectedOverride_conf,
		},
		{
			Key:   "fluent-bit.config.script",
			Value: expectedFileScript,
		},
		{
			Key:   "fluent-bit.config.extra",
			Value: expectedOverride_config,
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

	operation := internal.UpgradeKymaOperation{
		InputCreator: inputCreatorMock,
		Operation: internal.Operation{

			ProvisioningParameters: internal.ProvisioningParameters{ErsContext: internal.ERSContext{SubAccountID: "1234567890"}},
			InstanceDetails: internal.InstanceDetails{
				Cls: internal.ClsData{
					Overrides: encrypted,
				},
			},
		},
		RuntimeVersion: internal.RuntimeVersionData{
			Version: "1.19",
			Origin:  "foo",
		},
	}
	repo.InsertUpgradeKymaOperation(operation)
	// when
	_, repeat, err := svc.Run(operation, logger.NewLogDummy())
	//then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
}

func TestClsUpgradeAuditLogStep_HappyPath_SeqHttp(t *testing.T) {
	// given
	mm := afero.NewMemMapFs()

	fileScript := `
func myScript() {
foo: sub_account_id
bar: tenant_id
return "fooBar"
}
`
	overridesIn := cls.OverrideParams{
		FluentdEndPoint: "foo.bar",
		FluentdPassword: "fooPass",
		FluentdUsername: "fooUser",
		KibanaUrl:       "Kiib.url",
	}
	secretKey := "1234567890123456"

	// when
	encrypted, err := cls.EncryptOverrides(secretKey, &overridesIn)
	assert.NoError(t, err)

	err = afero.WriteFile(mm, "/auditlog-script/script", []byte(fileScript), 0755)
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
	svc := NewClsUpgradeAuditLogOverridesStep(repo, cfg, "1234567890123456")
	svc.fs = mm

	inputCreatorMock := &automock.ProvisionerInputCreator{}
	defer inputCreatorMock.AssertExpectations(t)

	expectedOverride_conf := `
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
    HTTP_User        aaaa
    HTTP_Passwd      aaaa
    Format           json_stream
    tls              on`
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
    HTTP_User        aaaa
    HTTP_Passwd      aaaa
    Format           json_stream
    tls              on`
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
			Key:   "fluent-bit.conf.script",
			Value: expectedFileScript,
		},
		{
			Key:   "fluent-bit.conf.extra",
			Value: expectedOverride_conf,
		},
		{
			Key:   "fluent-bit.config.script",
			Value: expectedFileScript,
		},
		{
			Key:   "fluent-bit.config.extra",
			Value: expectedOverride_config,
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

	operation := internal.UpgradeKymaOperation{
		RuntimeVersion: internal.RuntimeVersionData{
			Version: "1.20",
			Origin:  "foo",
		},
		InputCreator: inputCreatorMock,
		Operation: internal.Operation{
			ProvisioningParameters: internal.ProvisioningParameters{ErsContext: internal.ERSContext{SubAccountID: "1234567890"}},
			InstanceDetails: internal.InstanceDetails{
				Cls: internal.ClsData{
					Overrides: encrypted,
				},
			},
		},
	}
	repo.InsertUpgradeKymaOperation(operation)
	// when
	_, repeat, err := svc.Run(operation, logger.NewLogDummy())
	//then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
}
