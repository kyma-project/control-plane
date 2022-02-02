package broker

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestHideSensitiveDataFromContext(t *testing.T) {
	// This test is just a way to show, how the hideSensitiveDataFromContext works
	in := map[string]interface{}{
		"password": "pa2345",
		"username": "johnsmith",
		"subobject": map[string]interface{}{
			"secret": "val",
		},
		"isValid": true,
	}

	// when
	out := hideSensitiveDataFromContext(in)

	// then
	fmt.Println(out)

	d, _ := json.Marshal(out)
	fmt.Println(string(d))
}
