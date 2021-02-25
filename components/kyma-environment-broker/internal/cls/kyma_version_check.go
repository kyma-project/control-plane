package cls

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
)

func IsKymaVersion_1_20(runTimeVersion string) (bool, error) {
	c, err := semver.NewConstraint("<1.20.x")
	if err != nil {
		return false, errors.New("unable to parse constraint for kyma version %s to set correct fluent bit plugin")
	}
	pr, err := regexp.Compile("PR")
	if err != nil {
		return false, errors.New("unable to compile regex 'pr'")
	}
	master, err := regexp.Compile("master")
	if err != nil {
		return false, errors.New("unable to compile regex 'master'")
	}

	version, err := semver.NewVersion(runTimeVersion)
	if err != nil {
		if pr.MatchString(runTimeVersion) || master.MatchString(runTimeVersion) {
			return true, nil
		}
		return false, fmt.Errorf("unable to parse kyma version %s to set correct fluent bit plugin", runTimeVersion)
	}

	check := c.Check(version)
	if check {
		return false, nil
	} else {
		return true, nil
	}
}
