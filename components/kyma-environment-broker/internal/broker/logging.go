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
	err := json.Unmarshal(d, &data)
	if err != nil {
		return map[string]interface{}{}
	}
	for k, v := range data {
		switch reflect.TypeOf(v).Kind() {
		case reflect.String:
			if _, exists := openKeys[k]; !exists {
				data[k] = "*****"
			}
		case reflect.Map:
			data[k] = hideSensitiveDataFromContext(v.(map[string]interface{}))
		}
	}

	return data
}

func hideSensitiveDataFromContext(input map[string]interface{}) map[string]interface{} {
	for k, v := range input {
		if reflect.TypeOf(v).Kind() == reflect.String {
			if _, exists := openKeys[k]; !exists {
				input[k] = "*****"
			}
		}
		if reflect.TypeOf(v).Kind() == reflect.Map {
			input[k] = hideSensitiveDataFromContext(v.(map[string]interface{}))
		}
	}

	return input
}

func marshallRawContext(d map[string]interface{}) string {
	b, err := json.Marshal(d)
	if err != nil {
		return fmt.Sprintf("unable to marshal context data")
	}
	return string(b)
}
