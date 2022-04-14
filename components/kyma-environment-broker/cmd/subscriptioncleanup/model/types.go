package model

import "github.com/pkg/errors"

type HyperscalerType string

const (
	GCP   HyperscalerType = "gcp"
	Azure HyperscalerType = "azure"
	AWS   HyperscalerType = "aws"
)

func NewHyperscalerType(provider string) (HyperscalerType, error) {

	hyperscalerType := HyperscalerType(provider)

	switch hyperscalerType {
	case GCP, Azure, AWS:
		return hyperscalerType, nil
	}
	return "", errors.Errorf("unknown Hyperscaler provider type: %s", provider)
}
