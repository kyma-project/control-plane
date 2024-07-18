package util

import (
	"os"
)

const pvMountPath = "/testdata/provisioner"

func WriteToPV(requestData, shootName string) error {
	file, err := os.OpenFile(pvMountPath+"/"+shootName+".txt", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	if _, err = file.Write([]byte(requestData)); err != nil {
		file.Close()
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}
	return nil
}
