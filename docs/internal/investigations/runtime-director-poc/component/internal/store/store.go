package store

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/kyma-project/control-plane/docs/internal/investigations/runtime-director-poc/component/pkg/apperror"
	"github.com/kyma-project/control-plane/docs/internal/investigations/runtime-director-poc/component/pkg/model"
)

type storeConfig struct {
	Runtimes []model.Runtime `json:"runtimes"`
}

type Store struct {
	filePath string
	cached   storeConfig
}

func New(filePath string) *Store {
	return &Store{
		filePath: filePath,
	}
}

func (s *Store) GetRuntimeByID(id string) (model.Runtime, error) {
	for _, rtm := range s.cached.Runtimes {
		if rtm.ID != id {
			continue
		}

		return rtm, nil
	}

	return model.Runtime{}, apperror.NotFoundError
}

func (s *Store) ListRuntimes() ([]model.Runtime, error) {
	return s.cached.Runtimes, nil
}

func (s *Store) LoadConfig() error {
	absPath, err := filepath.Abs(s.filePath)
	if err != nil {
		return fmt.Errorf("while constructing absolute path from '%s': %w", s.filePath, err)
	}
	bytes, err := ioutil.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("while reading file from path '%s': %w", absPath, err)
	}

	var cfg storeConfig
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return fmt.Errorf("while unmarshaling yaml file: %w", err)
	}

	s.cached = cfg
	return nil
}
