package cleaner

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func Halt() error {
	value, exists := os.LookupEnv("DATABASE_EMBEDDED")
	var err error = nil
	if exists && value == "false" {
		err = HaltCloudSqlProxy()
	} else if exists && value == "true" {
		err = HaltIstioSidecar()
	}

	return err
}

func HaltCloudSqlProxy() error {
	fmt.Println("# HALT CLOUD SQL PROXY #")
	matches, err := filepath.Glob("/proc/*/comm")
	if err != nil {
		return fmt.Errorf("while reading process list: %s", err)
	}

	if len(matches) == 0 {
		fmt.Println("No matching processes found")
	}

	for _, file := range matches {

		target, _ := os.ReadFile(file)

		if len(target) > 0 && strings.Contains(string(target), "cloud-sql-proxy") {
			splitted := strings.Split(file, "/")

			pid, err := strconv.Atoi(splitted[2])
			if err != nil {
				return fmt.Errorf("while reading process id: %s", err)
			}

			proc, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("while reading process by pid: %s", err)
			}

			err = proc.Signal(syscall.SIGTERM)
			if err != nil {
				return fmt.Errorf("while killing cloud-sql-proxy: %s", err)
			}
			break
		}
	}
	if len(matches) == 0 {
		fmt.Println("No cloud-sql-proxy process found")
	}
	return nil
}

func HaltIstioSidecar() error {
	fmt.Println("# HALT ISTIO SIDECAR #")
	resp, err := http.PostForm("http://127.0.0.1:15020/quitquitquit", url.Values{})

	if err != nil {
		return fmt.Errorf("while sending post to quit Istio sidecar: %s", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		fmt.Printf("Quiting istio, response status is: %d", resp.StatusCode)
		return nil
	}

	return nil
}
