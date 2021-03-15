package cls

import (
	"fmt"

	"github.com/Masterminds/semver"
)

func IsKymaVersionAtLeast_1_20(runTimeVersion string) (bool, error) {
	c, err := semver.NewConstraint("<1.20.x")
	if err != nil {
		return false, fmt.Errorf("unable to parse constraint for Kyma version %s to set respective Fluent Bit plugin", runTimeVersion)
	}

	version, err := semver.NewVersion(runTimeVersion)
	if err != nil {
		// Return here if get some non semver image version.
		return true, nil
	}

	return !c.Check(version), nil
}
