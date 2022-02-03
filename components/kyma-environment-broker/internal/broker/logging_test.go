package broker

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHideSensitiveDataFromContext(t *testing.T) {
	// This test is just a way to show, how the hideSensitiveDataFromContext works
	in := map[string]interface{}{
		"password": "pa2345",
		"username": "johnsmith",
		"subobject": map[string]interface{}{
			"secret": "val",
			"sm_url": "https://sm.url.com",
		},
		"isValid": true,
	}

	// when
	out := hideSensitiveDataFromContext(in)

	d, err := json.Marshal(out)
	require.NoError(t, err)
	assert.Equal(t, `{"isValid":true,"password":"*****","subobject":{"secret":"*****","sm_url":"https://sm.url.com"},"username":"*****"}`, string(d))
}
