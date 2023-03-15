package euaccess

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	WhitelistKey = "whitelist"
)

type WhitelistSet map[string]struct{}

func IsNotWhitelisted(globalAccountId string, whitelist WhitelistSet) bool {
	_, found := whitelist[globalAccountId]
	return !found
}

func ReadWhitelistedGlobalAccountIdsFromFile(filename string) (WhitelistSet, error) {
	yamlData := make(map[string][]string)
	whitelistSet := WhitelistSet{}
	var whitelist, err = os.ReadFile(filename)
	if err != nil {
		return whitelistSet, fmt.Errorf("while reading %s file with whitelisted GlobalAccountIds config: %w", filename, err)
	}
	err = yaml.Unmarshal(whitelist, &yamlData)
	if err != nil {
		return whitelistSet, fmt.Errorf("while unmarshalling a file with whitelisted GlobalAccountIds config: %w", err)
	}
	for _, id := range yamlData[WhitelistKey] {
		whitelistSet[id] = struct{}{}
	}
	return whitelistSet, nil
}
