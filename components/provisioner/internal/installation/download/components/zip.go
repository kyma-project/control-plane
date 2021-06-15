package components

import (
	"os"

	"github.com/kyma-incubator/hydroform/parallel-install/pkg/archive"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/download"

	"github.com/pkg/errors"
)

func DownloadZip(URL string, tmpDst string) error {
	path, err := download.GetFile(URL, tmpDst)
	if err != nil {
		return errors.Wrap(err, "while downloading zip archive")
	}

	err = archive.Unzip(path, tmpDst)
	if err != nil {
		return errors.Wrap(err, "while unzipping zip archive")
	}

	err = os.Remove(path)
	if err != nil {
		return errors.Wrap(err, "while removing zip archive")
	}

	return nil
}
