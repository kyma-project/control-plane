package avs

import (
	"fmt"
	"os"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"gopkg.in/yaml.v3"
)

type Config struct {
	OauthTokenEndpoint                              string
	OauthUsername                                   string
	OauthPassword                                   string
	OauthClientId                                   string
	ApiEndpoint                                     string
	DefinitionType                                  string `envconfig:"default=BASIC"`
	Disabled                                        bool   `envconfig:"default=false"`
	InternalTesterAccessId                          int64
	InternalTesterService                           string `envconfig:"optional"`
	InternalTesterTags                              []*Tag `envconfig:"optional"`
	GroupId                                         int64
	ExternalTesterAccessId                          int64
	ExternalTesterService                           string `envconfig:"optional"`
	ExternalTesterTags                              []*Tag `envconfig:"optional"`
	ParentId                                        int64
	AdditionalTagsEnabled                           bool
	GardenerShootNameTagClassId                     int
	GardenerSeedNameTagClassId                      int
	RegionTagClassId                                int
	TrialInternalTesterAccessId                     int64    `envconfig:"optional"`
	TrialParentId                                   int64    `envconfig:"optional"`
	TrialGroupId                                    int64    `envconfig:"optional"`
	MaintenanceModeDuringUpgradeDisabled            bool     `envconfig:"default=false"`
	MaintenanceModeDuringUpgradeAlwaysDisabledGAIDs []string `envconfig:"-"`
}

func (c *Config) IsTrialConfigured() bool {
	return c.TrialInternalTesterAccessId != 0 && c.TrialParentId != 0 && c.TrialGroupId != 0
}

func (c *Config) ReadMaintenanceModeDuringUpgradeAlwaysDisabledGAIDsFromYaml(yamlFilePath string) error {
	yamlData, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return fmt.Errorf("while reading YAML file with GA IDs: %w", err)
	}
	var gaIDs struct {
		MaintenanceModeDuringUpgradeAlwaysDisabledGAIDs []string `yaml:"maintenanceModeDuringUpgradeAlwaysDisabledGAIDs"`
	}
	err = yaml.Unmarshal(yamlData, &gaIDs)
	if err != nil {
		return fmt.Errorf("while unmarshalling YAML file with GA IDs: %w", err)
	}

	c.MaintenanceModeDuringUpgradeAlwaysDisabledGAIDs = append(
		c.MaintenanceModeDuringUpgradeAlwaysDisabledGAIDs,
		gaIDs.MaintenanceModeDuringUpgradeAlwaysDisabledGAIDs...)

	return nil
}

type avsError struct {
	message string
}

func (e avsError) Error() string {
	return e.message
}

func (e avsError) Component() kebError.ErrComponent {
	return kebError.ErrAVS
}

func (e avsError) Reason() kebError.ErrReason {
	return kebError.ErrHttpStatusCode
}

func NewAvsError(format string, args ...interface{}) kebError.ErrorReporter {
	return avsError{message: fmt.Sprintf(format, args...)}
}
