package uaa

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	clusterDomain = "uaa-test.kyma-dev.shoot.canary.k8s-hana.ondemand.com"
	randomSuffix  = "_lxfaa"
)

func TestParametersBuilder_Generate(t *testing.T) {

	t.Run("parameters for the new instance", func(t *testing.T) {
		// Given
		pb := NewParametersFactory(Config{
			DeveloperGroup:      "runtimeDeveloper",
			DeveloperRole:       "KymaRuntimeDeveloper",
			NamespaceAdminGroup: "runtimeNamespaceAdmin",
			NamespaceAdminRole:  "KymaRuntimeNamespaceAdmin",
		})

		// When
		parameters := pb.Generate("uaa-test", clusterDomain, "xs-ap-name")

		// Then
		assertParameters(t, parameters)
	})
}

func assertParameters(t *testing.T, schema Schema) {
	roles := []string{"KymaRuntimeDeveloper_uaa_test", "KymaRuntimeNamespaceAdmin_uaa_test"}
	rolesWithName := []string{"$XSAPPNAME.KymaRuntimeDeveloper_uaa_test", "$XSAPPNAME.KymaRuntimeNamespaceAdmin_uaa_test"}
	groups := []string{"$XSAPPNAME.runtimeDeveloper", "$XSAPPNAME.runtimeNamespaceAdmin"}
	scopesGroup := []string{"$XSAPPNAME.email", "$XSAPPNAME.runtimeDeveloper", "$XSAPPNAME.runtimeNamespaceAdmin"}

	require.Equal(t, schema.TenantMode, "shared")

	require.Len(t, schema.Scopes, 3)
	names := make([]string, 0)
	for _, s := range schema.Scopes {
		names = append(names, s.Name)
	}
	require.ElementsMatch(t, scopesGroup, names)

	require.Len(t, schema.Authorities, 1)
	require.Equal(t, schema.Authorities[0], "$ACCEPT_GRANTED_AUTHORITIES")

	require.Len(t, schema.Oauth2Configuration.RedirectUris, 1)
	require.Equal(t, schema.Oauth2Configuration.RedirectUris[0], fmt.Sprintf("https://dex.%s/callback", clusterDomain))

	require.Len(t, schema.RoleTemplates, 2)
	require.Contains(t, roles, schema.RoleTemplates[0].Name)
	require.Contains(t, roles, schema.RoleTemplates[1].Name)
	require.Len(t, schema.RoleTemplates[0].ScopeReferences, 1)
	require.Len(t, schema.RoleTemplates[1].ScopeReferences, 1)
	require.Contains(t, groups, schema.RoleTemplates[0].ScopeReferences[0])
	require.Contains(t, groups, schema.RoleTemplates[1].ScopeReferences[0])

	require.Len(t, schema.RoleCollections, 2)
	require.Contains(t, roles, schema.RoleCollections[0].Name)
	require.Contains(t, roles, schema.RoleCollections[1].Name)
	require.Len(t, schema.RoleCollections[0].RoleTemplateReference, 1)
	require.Len(t, schema.RoleCollections[1].RoleTemplateReference, 1)
	require.Contains(t, rolesWithName, schema.RoleCollections[0].RoleTemplateReference[0])
	require.Contains(t, rolesWithName, schema.RoleCollections[1].RoleTemplateReference[0])
}

func parameters() string {
	return fmt.Sprintf(
		`{
  "authorities": [
    "$ACCEPT_GRANTED_AUTHORITIES"
  ],
  "oauth2-configuration": {
    "redirect-uris": [
      "https://dex.%s/callback"
    ],
    "system-attributes": [
      "groups",
      "rolecollections"
    ]
  },
  "role-templates": [
    {
      "description": "Runtime developer access to all managed resources",
      "name": "KymaRuntimeNamespaceDeveloper",
      "scope-references": [
        "$XSAPPNAME.runtimeDeveloper"
      ]
    },
    {
      "description": "Runtime admin access to all managed resources",
      "name": "KymaRuntimeNamespaceAdmin",
      "scope-references": [
        "$XSAPPNAME.runtimeNamespaceAdmin"
      ]
    }
  ],
  "scopes": [
    {
      "description": "get user email",
      "name": "$XSAPPNAME.email"
    },
    {
      "description": "Runtime developer access to all managed resources",
      "name": "$XSAPPNAME.runtimeDeveloper"
    },
    {
      "description": "Runtime admin access to all managed resources",
      "name": "$XSAPPNAME.runtimeNamespaceAdmin"
    }
  ],
  "tenant-mode": "shared",
  "xsappname": "%s%s"
}`, clusterDomain, strings.ReplaceAll(clusterDomain, ".", "_"), randomSuffix)
}
