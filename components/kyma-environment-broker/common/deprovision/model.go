package deprovision

import (
	"context"

	"golang.org/x/oauth2"
)

const (
	InstanceIdParam = "instance_id"
	ShootParam      = "shoot"
)

type DeprovisionParameters struct {
	ClientID           string
	ClientSecret       string
	TokenURL           string
	Scopes             []string
	AuthStyle          oauth2.AuthStyle
	EndpointURL        string
	Shoot              string
	InstanceID         string
	Context            context.Context
	Oauth2IssuerURL    string
	Oauth2ClientID     string
	Oauth2ClientSecret string
}
