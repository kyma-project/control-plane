package avs

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
