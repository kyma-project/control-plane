package provisioner

import (
	"bytes"
	"encoding/json"
	"reflect"
	"text/template"

	"github.com/sirupsen/logrus"

	"fmt"

	"strconv"

	"github.com/Masterminds/sprig"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
)

// Graphqlizer is responsible for converting Go objects to input arguments in graphql format
type Graphqlizer struct{}

func (g *Graphqlizer) ProvisionRuntimeInputToGraphQL(in gqlschema.ProvisionRuntimeInput) (string, error) {
	return g.genericToGraphQL(in, `{
		{{- if .RuntimeInput }}
      	runtimeInput: {{ RuntimeInputToGraphQL .RuntimeInput }},
		{{- end }}
		{{- if .ClusterConfig }}
		clusterConfig: {{ ClusterConfigToGraphQL .ClusterConfig }},
		{{- end }}
		{{- if .KymaConfig }}
		kymaConfig: {{ KymaConfigToGraphQL .KymaConfig }},
		{{- end }}
	}`)
}

func (g *Graphqlizer) RuntimeInputToGraphQL(in gqlschema.RuntimeInput) (string, error) {
	return g.genericToGraphQL(in, `{
		name: "{{.Name}}",
		{{- if .Description }}
		description: "{{.Description}}",
		{{- end }}
		{{- if .Labels }}
		labels: {{ LabelsToGQL .Labels}},
		{{- end }}
	}`)
}

func (g *Graphqlizer) LabelsToGQL(in gqlschema.Labels) (string, error) {
	return g.marshal(in), nil
}

func (g *Graphqlizer) ClusterConfigToGraphQL(in gqlschema.ClusterConfigInput) (string, error) {
	return g.genericToGraphQL(in, `{
		{{- if .GardenerConfig }}
		gardenerConfig: {{ GardenerConfigInputToGraphQL .GardenerConfig }},
		{{- end }}
		{{- if .Administrators }}
		administrators: {{.Administrators | marshal }},
		{{- end }}
	}`)
}

func (g *Graphqlizer) GardenerConfigInputToGraphQL(in gqlschema.GardenerConfigInput) (string, error) {
	return g.genericToGraphQL(in, `{
		{{- if .Name }}
		name: "{{.Name}}",
        {{- end }}
		kubernetesVersion: "{{.KubernetesVersion}}",
        {{- if .VolumeSizeGb }}
		volumeSizeGB: {{.VolumeSizeGb }},
        {{- end }}
		machineType: "{{.MachineType}}",
		{{- if .MachineImage }}
		machineImage: "{{.MachineImage}}",
		{{- end}}
		{{- if .MachineImageVersion }}
		machineImageVersion: "{{.MachineImageVersion}}",
		{{- end }}
		region: "{{.Region}}",
		provider: "{{ .Provider }}",
		{{- if .Purpose }}
		purpose: "{{ .Purpose }}",
		{{- end }}
		{{- if .LicenceType }}
		licenceType: "{{ .LicenceType }}",
		{{- end }}
        {{- if .DiskType }}
		diskType: "{{.DiskType}}",
        {{- end }}
		targetSecret: "{{ .TargetSecret }}",
		workerCidr: "{{ .WorkerCidr }}",
        autoScalerMin: {{ .AutoScalerMin }},
        autoScalerMax: {{ .AutoScalerMax }},
        maxSurge: {{ .MaxSurge }},
		maxUnavailable: {{ .MaxUnavailable }},
		{{- if .EnableKubernetesVersionAutoUpdate }}
		enableKubernetesVersionAutoUpdate: {{ .EnableKubernetesVersionAutoUpdate }},
		{{- end }}
		{{- if .EnableMachineImageVersionAutoUpdate }}
		enableMachineImageVersionAutoUpdate: {{ .EnableMachineImageVersionAutoUpdate }},
		{{- end }}
		{{- if .ProviderSpecificConfig }}
		providerSpecificConfig: {
			{{- if .ProviderSpecificConfig.AzureConfig }}
			azureConfig: {{ AzureProviderConfigInputToGraphQL .ProviderSpecificConfig.AzureConfig }},
			{{- end}}
			{{- if .ProviderSpecificConfig.GcpConfig }}
			gcpConfig: {{ GCPProviderConfigInputToGraphQL .ProviderSpecificConfig.GcpConfig }},
			{{- end}}
            {{- if .ProviderSpecificConfig.AwsConfig }}
			awsConfig: {{ AWSProviderConfigInputToGraphQL .ProviderSpecificConfig.AwsConfig }},
			{{- end}}
            {{- if .ProviderSpecificConfig.OpenStackConfig }}
			openStackConfig: {{ OpenStackProviderConfigInputToGraphQL .ProviderSpecificConfig.OpenStackConfig }},
			{{- end}}
        }
		{{- end}}
        {{- if .OidcConfig }}
        oidcConfig: {
            clientID: "{{ .OidcConfig.ClientID }}",
            issuerURL: "{{ .OidcConfig.IssuerURL }}",
            groupsClaim: "{{ .OidcConfig.GroupsClaim }}",
            signingAlgs: {{ .OidcConfig.SigningAlgs | marshal }},
            usernameClaim: "{{ .OidcConfig.UsernameClaim }}",
            usernamePrefix: "{{ .OidcConfig.UsernamePrefix }}",
        }
        {{- end }}
        {{- if .DNSConfig }}
        dnsConfig: {
            {{- with .DNSConfig.Providers }}
            providers: [
                {{- range . }}
                {
                    domainsInclude: {{ .DomainsInclude | marshal }},
                    primary: {{ .Primary }},
                    secretName: {{ .SecretName | strQuote }},
                    type: {{ .Type | strQuote }},
                }
                {{- end }}
            ]
            {{- end }}
        }
        {{- end }}
    }`)
}

func (g *Graphqlizer) AzureProviderConfigInputToGraphQL(in gqlschema.AzureProviderConfigInput) (string, error) {
	return g.genericToGraphQL(in, `{
		vnetCidr: "{{.VnetCidr}}",
		{{- if .Zones }}
		zones: {{.Zones | marshal }},
		{{- end }}
	}`)
}

func (g *Graphqlizer) GCPProviderConfigInputToGraphQL(in gqlschema.GCPProviderConfigInput) (string, error) {
	return fmt.Sprintf(`{ zones: %s }`, g.marshal(in.Zones)), nil
}

func (g *Graphqlizer) AWSProviderConfigInputToGraphQL(in gqlschema.AWSProviderConfigInput) (string, error) {
	return g.genericToGraphQL(in, `{
		vpcCidr: "{{.VpcCidr}}",
		{{- with .AwsZones }}
		awsZones: [
		  {{- range . }}
		  {
			name: "{{ .Name }}",
			workerCidr: "{{ .WorkerCidr }}",
			publicCidr: "{{ .PublicCidr }}",
			internalCidr: "{{ .InternalCidr }}",
		  }
		  {{- end }}
		]
		{{- end }}
	}`)
}

func (g *Graphqlizer) OpenStackProviderConfigInputToGraphQL(in gqlschema.OpenStackProviderConfigInput) (string, error) {
	return fmt.Sprintf(`{
		zones: %s,
		floatingPoolName: "%s",
		cloudProfileName: "%s",
		loadBalancerProvider: "%s"
}`, g.marshal(in.Zones), in.FloatingPoolName, in.CloudProfileName, in.LoadBalancerProvider), nil
}

func (g *Graphqlizer) KymaConfigToGraphQL(in gqlschema.KymaConfigInput) (string, error) {
	return g.genericToGraphQL(in, `{
		version: "{{ .Version }}",
		{{- if .Profile }}
		profile: {{ .Profile }},
		{{- end }}
		{{- if .ConflictStrategy }}
		conflictStrategy: {{ .ConflictStrategy }},
		{{- end }}
		{{- with .Components }}
        components: [
		  {{- range . }}
          {
            component: "{{ .Component }}",
            namespace: "{{ .Namespace }}",
            {{- if .SourceURL }}
            sourceURL: "{{ .SourceURL }}",
            {{- end }}
			{{- if .ConflictStrategy }}
			conflictStrategy: {{ .ConflictStrategy }},
			{{- end }}
      	    {{- with .Configuration }}
            configuration: [
			  {{- range . }}
              {
                key: "{{ .Key }}",
                value: {{ .Value | strQuote }},
				{{- if .Secret }}
                secret: true,
				{{- end }}
              }
		      {{- end }}
            ]
		    {{- end }}
          }
		  {{- end }}
        ]
      	{{- end }}
		{{- with .Configuration }}
		configuration: [
		  {{- range . }}
		  {
			key: "{{ .Key }}",
			value: {{ .Value | strQuote }},
			{{- if .Secret }}
			secret: true,
			{{- end }}
		  }
		  {{- end }}
		]
		{{- end }}
	}`)
}

func (g *Graphqlizer) GardenerUpgradeInputToGraphQL(in gqlschema.GardenerUpgradeInput) (string, error) {
	return g.genericToGraphQL(in, `{
      {{- if .KubernetesVersion }}
      kubernetesVersion: "{{.KubernetesVersion}}",
      {{- end }}
      {{- if .MachineImage }}
      machineImage: "{{.MachineImage}}",
      {{- end}}
      {{- if .MachineImageVersion }}
      machineImageVersion: "{{.MachineImageVersion}}",
      {{- end }}
      {{- if .AutoScalerMin }}
      autoScalerMin: {{ .AutoScalerMin }},
      {{- end }}
      {{- if .AutoScalerMax }}
      autoScalerMax: {{ .AutoScalerMax }},
      {{- end }}
      {{- if .MaxSurge }}
      maxSurge: {{ .MaxSurge }},
      {{- end }}
      {{- if .MaxUnavailable }}
      maxUnavailable: {{ .MaxUnavailable }},
      {{- end }}
      {{- if .EnableKubernetesVersionAutoUpdate }}
      enableKubernetesVersionAutoUpdate: {{ .EnableKubernetesVersionAutoUpdate }},
      {{- end }}
      {{- if .EnableMachineImageVersionAutoUpdate }}
      enableMachineImageVersionAutoUpdate: {{ .EnableMachineImageVersionAutoUpdate }},
      {{- end }}
      {{- if .OidcConfig }}
      oidcConfig: {
        clientID: "{{ .OidcConfig.ClientID }}",
        issuerURL: "{{ .OidcConfig.IssuerURL }}",
        groupsClaim: "{{ .OidcConfig.GroupsClaim }}",
        signingAlgs: {{ .OidcConfig.SigningAlgs | marshal }},
        usernameClaim: "{{ .OidcConfig.UsernameClaim }}",
        usernamePrefix: "{{ .OidcConfig.UsernamePrefix }}",
      },
      {{- end }}
    }`)
}

func (g *Graphqlizer) marshal(obj interface{}) string {
	var out string

	val := reflect.ValueOf(obj)

	switch val.Kind() {
	case reflect.Map:
		s, err := g.genericToGraphQL(obj, `{ {{- range $k, $v := . }}{{ $k }}:{{ marshal $v }},{{ end -}} }`)
		if err != nil {
			logrus.Warnf("failed to marshal labels: %s", err.Error())
			return ""
		}
		out = s
	case reflect.Slice, reflect.Array:
		s, err := g.genericToGraphQL(obj, `[{{ range $i, $e := . }}{{ if $i }},{{ end }}{{ marshal $e }}{{ end }}]`)
		if err != nil {
			logrus.Warnf("failed to marshal labels: %s", err.Error())
			return ""
		}
		out = s
	default:
		marshalled, err := json.Marshal(obj)
		if err != nil {
			logrus.Warnf("failed to marshal labels: %s", err.Error())
			return ""
		}
		out = string(marshalled)
	}

	return out
}

func (g *Graphqlizer) UpgradeRuntimeInputToGraphQL(in gqlschema.UpgradeRuntimeInput) (string, error) {
	return g.genericToGraphQL(in, `{
		kymaConfig: {{ KymaConfigToGraphQL .KymaConfig }}
	}`)
}

func (g Graphqlizer) UpgradeShootInputToGraphQL(in gqlschema.UpgradeShootInput) (string, error) {
	return g.genericToGraphQL(in, `{
    gardenerConfig: {{ GardenerUpgradeInputToGraphQL .GardenerConfig }},
    {{- if .Administrators }}
    administrators: {{.Administrators | marshal }},
    {{- end }}
}`)
}

func (g *Graphqlizer) genericToGraphQL(obj interface{}, tmpl string) (string, error) {
	fm := sprig.TxtFuncMap()
	fm["marshal"] = g.marshal
	fm["RuntimeInputToGraphQL"] = g.RuntimeInputToGraphQL
	fm["ClusterConfigToGraphQL"] = g.ClusterConfigToGraphQL
	fm["KymaConfigToGraphQL"] = g.KymaConfigToGraphQL
	fm["GardenerConfigInputToGraphQL"] = g.GardenerConfigInputToGraphQL
	fm["GardenerUpgradeInputToGraphQL"] = g.GardenerUpgradeInputToGraphQL
	fm["AzureProviderConfigInputToGraphQL"] = g.AzureProviderConfigInputToGraphQL
	fm["GCPProviderConfigInputToGraphQL"] = g.GCPProviderConfigInputToGraphQL
	fm["AWSProviderConfigInputToGraphQL"] = g.AWSProviderConfigInputToGraphQL
	fm["OpenStackProviderConfigInputToGraphQL"] = g.OpenStackProviderConfigInputToGraphQL
	fm["LabelsToGQL"] = g.LabelsToGQL
	fm["strQuote"] = strconv.Quote

	t, err := template.New("tmpl").Funcs(fm).Parse(tmpl)
	if err != nil {
		return "", errors.Wrapf(err, "while parsing template")
	}

	var b bytes.Buffer

	if err := t.Execute(&b, obj); err != nil {
		return "", errors.Wrap(err, "while executing template")
	}
	return b.String(), nil
}
