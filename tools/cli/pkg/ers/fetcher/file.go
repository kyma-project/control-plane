package fetcher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
)

type FileClient struct {
	filename string
}

func NewFileClient(filename string) InstanceFetcher {
	return &FileClient{
		filename: filename,
	}
}

func (c *FileClient) GetInstanceById(id string) (*ers.Instance, error) {
	instances, err := c.GetAllInstances()
	if err != nil {
		return &ers.Instance{}, fmt.Errorf("while loading specific instance: %w", err)
	}

	for _, inst := range instances {
		if inst.Id == id {
			return &inst, nil
		}
	}

	return &ers.Instance{}, fmt.Errorf("instance %s not found", id)
}

func (c *FileClient) GetAllInstances() ([]ers.Instance, error) {
	data, err := ioutil.ReadFile(c.filename)
	if err != nil {
		return []ers.Instance{}, fmt.Errorf("error while loading all instances: %w", err)
	}

	var objects []ers.Instance
	err = json.Unmarshal(data, &objects)
	if err != nil {
		return []ers.Instance{}, fmt.Errorf("error while unmarshaling read object: %w", err)
	}

	return objects, nil
}
