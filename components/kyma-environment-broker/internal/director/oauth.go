package director

import (
	"time"

	"golang.org/x/oauth2"
)

type dupa struct {
}

func (d *dupa) GetAuthorizationToken() (oauth2.Token, error) {

	oauth2.Config{
		ClientID:     "",
		ClientSecret: "",
		Endpoint:     oauth2.Endpoint{},
		RedirectURL:  "",
		Scopes:       nil,
	}
	kupa := oauth2.Token{
		AccessToken:  "",
		TokenType:    "",
		RefreshToken: "",
		Expiry:       time.Time{},
	}
}
