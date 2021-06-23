package uaa

import (
	"fmt"
	"math/rand"
	"strings"
)

type ParametersFactory struct {
	developerGroup string
	adminGroup     string
	developerRole  string
	adminRole      string
}

// Config holds configuration for the UAA domain
type Config struct {
	DeveloperGroup      string
	DeveloperRole       string
	NamespaceAdminGroup string
	NamespaceAdminRole  string
}

func NewParametersFactory(cfg Config) *ParametersFactory {
	return &ParametersFactory{
		developerGroup: cfg.DeveloperGroup,
		adminGroup:     cfg.NamespaceAdminGroup,
		developerRole:  cfg.DeveloperRole,
		adminRole:      cfg.NamespaceAdminRole,
	}
}

type Schema struct {
	Xsappname           string              `json:"xsappname"`
	TenantMode          string              `json:"tenant-mode"`
	Scopes              []Scope             `json:"scopes"`
	Authorities         []string            `json:"authorities"`
	RoleTemplates       []RoleTemplate      `json:"role-templates"`
	RoleCollections     []RoleCollection    `json:"role-collections"`
	Oauth2Configuration Oauth2Configuration `json:"oauth2-configuration"`
}

type Scope struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoleTemplate struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	ScopeReferences []string `json:"scope-references"`
}

type RoleCollection struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	RoleTemplateReference []string `json:"role-template-references"`
}

type Oauth2Configuration struct {
	RedirectUris     []string `json:"redirect-uris"`
	SystemAttributes []string `json:"system-attributes"`
}

func (pf *ParametersFactory) Generate(shootName, domain, xsappname string) Schema {
	devRole := roleName(pf.developerRole, shootName)
	adminRole := roleName(pf.adminRole, shootName)
	redirectURL := fmt.Sprintf("https://dex.%s/callback", strings.Trim(domain, "/"))
	parameters := Schema{
		Xsappname:  xsappname,
		TenantMode: "shared",
		Scopes: []Scope{
			{
				Name:        "$XSAPPNAME.email",
				Description: "get user email",
			},
			{
				Name:        fmt.Sprintf("$XSAPPNAME.%s", pf.developerGroup),
				Description: "Runtime developer access to all managed resources",
			},
			{
				Name:        fmt.Sprintf("$XSAPPNAME.%s", pf.adminGroup),
				Description: "Runtime admin access to all managed resources",
			},
		},
		Authorities: []string{
			"$ACCEPT_GRANTED_AUTHORITIES",
		},
		RoleTemplates: []RoleTemplate{
			{
				Name:        devRole,
				Description: "Runtime developer access to all managed resources",
				ScopeReferences: []string{
					fmt.Sprintf("$XSAPPNAME.%s", pf.developerGroup),
				},
			},
			{
				Name:        adminRole,
				Description: "Runtime admin access to all managed resources",
				ScopeReferences: []string{
					fmt.Sprintf("$XSAPPNAME.%s", pf.adminGroup),
				},
			},
		},
		RoleCollections: []RoleCollection{
			{
				Name:        devRole,
				Description: "Kyma Runtime Developer Role Collection for development tasks in given custom namespaces",
				RoleTemplateReference: []string{
					fmt.Sprintf("$XSAPPNAME.%s", devRole),
				},
			},
			{
				Name:        adminRole,
				Description: "Kyma Runtime Namespace Admin Role Collection for administration tasks across all custom namespaces",
				RoleTemplateReference: []string{
					fmt.Sprintf("$XSAPPNAME.%s", adminRole),
				},
			},
		},
		Oauth2Configuration: Oauth2Configuration{
			RedirectUris: []string{
				redirectURL,
			},
			SystemAttributes: []string{
				"groups",
				"rolecollections",
			},
		},
	}

	return parameters
}

func XSAppname(domain string) string {
	return fmt.Sprintf("%s_%s", strings.ReplaceAll(domain, ".", "_"), randomString(5))
}

func randomString(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyz")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// roleName creates proper name for RoleTemplates and RoleCollections
// according to SM the name may only include characters 'a'-'z', 'A'-'Z', '0'-'9', and '_'
func roleName(name, domain string) string {
	r := strings.NewReplacer(".", "_", ",", "_", ":", "", ";", "", "-", "_", "/", "", "\\", "")
	return fmt.Sprintf("%s_%s", r.Replace(name), r.Replace(domain))
}
