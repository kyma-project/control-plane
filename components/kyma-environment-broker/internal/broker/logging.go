package broker

import (
	"encoding/json"
	"fmt"
	"reflect"
)

var openKeys = map[string]struct{}{
	"sm_url":           {},
	"xsappname":        {},
	"globalaccount_id": {},
	"subaccount_id":    {},
}

func hideSensitiveDataFromRawContext(d []byte) map[string]interface{} {
	var data map[string]interface{}
	_ = json.Unmarshal(d, &data)
	return hideSensitiveDataFromContext(data)
}

func hideSensitiveDataFromContext(input map[string]interface{}) map[string]interface{} {
	copy := input

	for k, v := range copy {
		if reflect.TypeOf(v).Kind() == reflect.String {
			if _, exists := openKeys[k]; !exists {
				copy[k] = "*****"
			}
		}
		if reflect.TypeOf(v).Kind() == reflect.Map {
			copy[k] = hideSensitiveDataFromContext(v.(map[string]interface{}))
		}
	}

	return copy
}

func marshallRawContext(d map[string]interface{}) string {
	b, err := json.Marshal(d)
	if err != nil {
		return fmt.Sprintf("unable to marshall context data")
	}
	return string(b)
}
