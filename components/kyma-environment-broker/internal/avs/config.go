package avs

import (
	"fmt"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
)

type Config struct {
	OauthTokenEndpoint          string
	OauthUsername               string
	OauthPassword               string
	OauthClientId               string
	ApiEndpoint                 string
	DefinitionType              string `envconfig:"default=BASIC"`
	Disabled                    bool   `envconfig:"default=false"`
	InternalTesterAccessId      int64
	InternalTesterService       string `envconfig:"optional"`
	InternalTesterTags          []*Tag `envconfig:"optional"`
	GroupId                     int64
	ExternalTesterAccessId      int64
	ExternalTesterService       string `envconfig:"optional"`
	ExternalTesterTags          []*Tag `envconfig:"optional"`
	ParentId                    int64
	AdditionalTagsEnabled       bool
	GardenerShootNameTagClassId int
	GardenerSeedNameTagClassId  int
	RegionTagClassId            int
	TrialInternalTesterAccessId int64 `envconfig:"optional"`
	TrialParentId               int64 `envconfig:"optional"`
	TrialGroupId                int64 `envconfig:"optional"`
}

func (c Config) IsTrialConfigured() bool {
	return c.TrialInternalTesterAccessId != 0 && c.TrialParentId != 0 && c.TrialGroupId != 0
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
