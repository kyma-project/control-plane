package broker

import (
	"encoding/json"
)

func Marshal(obj interface{}) []byte {
	if obj == nil {
		return []byte{}
	}
	bytes, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return bytes
}
