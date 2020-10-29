package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetAzureResourceName(t *testing.T) {
	tests := []struct {
		name             string
		givenName        string
		wantResourceName string
	}{
		{
			name:             "all lowercase and starts with digit",
			givenName:        "1a23238d-1b04-3a9c-c139-405b75796ceb",
			wantResourceName: "k1a23238d-1b04-3a9c-c139-405b75796ceb",
		},
		{
			name:             "all uppercase and starts with digit",
			givenName:        "1A23238D-1B04-3A9C-C139-405B75796CEB",
			wantResourceName: "k1a23238d-1b04-3a9c-c139-405b75796ceb",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := GetAzureResourceName(test.givenName)
			assert.Equal(t, test.wantResourceName, got)
		})
	}
}
