package version

import (
	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
)

func IsKymaVersionAtLeast_1_20(runTimeVersion string) (bool, error) {
	return isKymaVersionAtLeast("<1.20.x", runTimeVersion)
}

func IsKymaVersionAtLeast_1_21(runTimeVersion string) (bool, error) {
	return isKymaVersionAtLeast("<1.21.x", runTimeVersion)
}

func isKymaVersionAtLeast(constraint, runTimeVersion string) (bool, error) {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, errors.Errorf("unable to parse constraint  %s for kyma version %s", constraint, runTimeVersion)
	}

	version, err := semver.NewVersion(runTimeVersion)
	if err != nil {
		// Return here if get some non semver image version.
		return true, nil
	}

	return !c.Check(version), nil
}
