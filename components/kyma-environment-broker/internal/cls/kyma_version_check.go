package cls

import (
	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
)

func IsKymaVersion_1_20(runTimeVersion string) (bool, error) {
	c, err := semver.NewConstraint("<1.20.x")
	if err != nil {
		return false, errors.New("unable to parse constraint for kyma version %s to set correct fluent bit plugin")
	}

	version, err := semver.NewVersion(runTimeVersion)
	if err != nil {
		// Return here if get some non semver image version.
		return true, nil
	}

	check := c.Check(version)
	if check {
		return false, nil
	} else {
		return true, nil
	}
}
