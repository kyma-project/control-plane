package metadata

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
)

const metadataFolderName = "metadata"

type Storage struct {
}

func (s *Storage) Save(metadata ers.MigrationMetadata) error {
	//Create dir output using above code
	if _, err := os.Stat(metadataFolderName); os.IsNotExist(err) {
		os.Mkdir(metadataFolderName, 0777)
	}
	data, err := json.MarshalIndent(metadata, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName(metadata.Id), data, 0644)
}

func fileName(id string) string {
	return fmt.Sprintf("%s/%s.json", metadataFolderName, id)
}

// Get reads existing metadata from a file or returns "empty" (zero-valued) metadata
func (s *Storage) Get(id string) (ers.MigrationMetadata, error) {
	var metadata ers.MigrationMetadata
	metadata.Id = id
	data, err := ioutil.ReadFile(fileName(id))
	if os.IsNotExist(err) {
		return metadata, nil
	}
	err = json.Unmarshal(data, &metadata)
	return metadata, err
}
