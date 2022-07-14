package config

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const requiredFields = "additional-components"

func Validate(cfgString string) error {
	reqs := strings.Split(requiredFields, ",")
	keys := getKeysFromConfigString(cfgString)
	sort.Strings(reqs)
	sort.Strings(keys)

	var missingConfigs []string
	keysString := strings.Join(keys, ",")
	for _, req := range reqs {
		if !strings.Contains(keysString, req) {
			missingConfigs = append(missingConfigs, req)
		}
	}

	if len(missingConfigs) > 0 {
		return fmt.Errorf("missing required configuration entires: %s", strings.Join(missingConfigs, ","))
	}
	return nil
}

func getKeysFromConfigString(cfgString string) []string {
	keys := make([]string, 0)
	s1 := strings.Split(cfgString, "\n")
	for _, entry := range s1 {
		r := []rune(entry)[0]
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		entry = strings.ReplaceAll(entry, " ", "")
		keys = append(keys, strings.Split(entry, ":")[0])
	}
	return keys
}
