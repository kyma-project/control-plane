package store

import (
	"fmt"
	"github.com/kyma-project/control-plane/poc/component/pkg/apperror"
	"github.com/kyma-project/control-plane/poc/component/pkg/model"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"path/filepath"
)

type storeConfig struct {
	Runtimes []model.Runtime `json:"runtimes"`
}

type Store struct {
	filePath string
}

func New(filePath string) *Store {
	s := &Store{
		filePath: filePath,
	}

	// verify is config can be properly loaded
	s.mustLoadConfig()
	return s
}

func (s *Store) GetRuntimeByID(id string) (model.Runtime, error) {
	cfg, err := s.loadConfig()
	if err != nil {
		return model.Runtime{}, fmt.Errorf("while loading config: %w", err)
	}

	for _, rtm := range cfg.Runtimes {
		if rtm.ID != id {
			continue
		}

		return rtm, nil
	}

	return model.Runtime{}, apperror.NotFoundError
}

func (s *Store) ListRuntimes() ([]model.Runtime, error) {
	cfg, err := s.loadConfig()
	if err != nil {
		return []model.Runtime{}, fmt.Errorf("while loading config: %w", err)
	}

	return cfg.Runtimes, nil
}

func (s *Store) loadConfig() (storeConfig, error) {
	absPath, err := filepath.Abs(s.filePath)
	if err != nil {
		return storeConfig{}, fmt.Errorf("while constructing absolute path from '%s': %w", s.filePath, err)
	}
	bytes, err := ioutil.ReadFile(absPath)
	if err != nil {
		return storeConfig{}, fmt.Errorf("while reading file from path '%s': %w", absPath, err)
	}

	var cfg storeConfig
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return storeConfig{}, fmt.Errorf("while unmarshaling yaml file: %w", err)
	}
	return cfg, nil
}

func (s *Store) mustLoadConfig() {
 	_, err := s.loadConfig()
 	if err != nil {
 		panic(err)
	}
}
