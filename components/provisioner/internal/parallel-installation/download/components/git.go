package components

import (
	"fmt"
	"os"
	"strings"

	"github.com/kyma-incubator/hydroform/parallel-install/pkg/git"

	"github.com/google/uuid"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

/**
The component supports the following urls:
- https://repository.com//path/to/component?ref=<revision>
- https://repository.com//path/to/component
- https://repository.com?ref=<revision>
- https://repository.com
*/
type GitComponent struct {
	url       string
	schema    string
	tmpPath   string
	dstPath   string
	src       string
	component string
	revision  string
}

func NewGitComponent(URL string, tmp, dst string) GitComponent {
	gc := GitComponent{tmpPath: tmp, dstPath: dst}

	gc.schema = "https://"
	switch {
	case strings.HasPrefix(URL, "https://"):
		gc.url = strings.TrimPrefix(URL, "https://")
	case strings.HasPrefix(URL, "http://"):
		gc.schema = "http://"
		gc.url = strings.TrimPrefix(URL, "http://")
	default:
		gc.url = URL
	}

	return gc
}

func (gc *GitComponent) DownloadGitComponent() error {
	err := gc.splitURL()
	if err != nil {
		return errors.Wrap(err, "while reading URL")
	}

	var repoPath string
	if gc.component == "" {
		repoPath = gc.dstPath
	} else {
		repoPath = fmt.Sprintf(gc.tmpPath, uuid.New().String())
	}

	// cloning repository with searching component
	err = git.CloneRepo(gc.src, repoPath, gc.revision)
	if err != nil {
		return errors.Wrapf(err, "while cloning repository for %s (revision: %s)", gc.src, gc.revision)
	}

	// if a concrete component is not specified then download process is end
	if gc.component == "" {
		return nil
	}

	// copy searching component to the destination path
	err = copy.Copy(fmt.Sprintf("%s/%s", repoPath, gc.component), gc.dstPath)
	if err != nil {
		return errors.Wrap(err, "while copying component to the destination path")
	}

	// removing unnecessary temporary repository files
	err = os.RemoveAll(repoPath)
	if err != nil {
		return errors.Wrap(err, "while removing repository temporary files")
	}

	return nil
}

func (gc *GitComponent) splitURL() error {
	parts := strings.Split(gc.url, "//")
	switch len(parts) {
	case 1:
		path, revision, err := splitRevision(gc.url)
		if err != nil {
			return errors.Wrap(err, "while splitting URL for revision")
		}
		gc.src = fmt.Sprintf("%s%s", gc.schema, path)
		gc.revision = revision
		return nil
	case 2:
		component, revision, err := splitRevision(parts[1])
		if err != nil {
			return errors.Wrap(err, "while splitting URL for revision")
		}
		gc.src = fmt.Sprintf("%s%s", gc.schema, parts[0])
		gc.component = component
		gc.revision = revision
		return nil
	}

	return errors.Errorf("unsupported URL: %s", gc.url)
}

func splitRevision(src string) (string, string, error) {
	parts := strings.Split(src, "?ref=")
	if len(parts) == 0 || len(parts) > 2 {
		return "", "", errors.Errorf("unsupported URL with revision: %v", parts)
	}
	if len(parts) == 1 {
		return parts[0], "", nil
	}
	return parts[0], parts[1], nil
}
