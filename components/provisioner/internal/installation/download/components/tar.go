package components

import (
	"os"

	"github.com/kyma-incubator/hydroform/parallel-install/pkg/archive"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/download"

	"github.com/pkg/errors"
)

func DownloadTgz(URL string, tmpDst string) error {
	path, err := download.GetFile(URL, tmpDst)
	if err != nil {
		return errors.Wrap(err, "while downloading tgz archive")
	}

	err = archive.Untar(path, tmpDst)
	if err != nil {
		return errors.Wrap(err, "while unzipping tgz archive")
	}

	err = os.Remove(path)
	if err != nil {
		return errors.Wrap(err, "while removing tgz archive")
	}

	return nil
}
