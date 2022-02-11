package command

import (
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
)

func NewInstancesCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "instances",
		Short: "Displays ERS instances.",
		Long:  `Displays most important information about ERS instances.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: this is an example of a call to ERS

			url := GlobalOpts.ErsUrl() + "provisioning/v1/kyma/environments?page=0&size=5"

			resp, err := ErsHttpClient.Get(url)
			if err != nil {
				fmt.Println("Request error: ", err)
				return err
			}
			defer func() {
				resp.Body.Close()
			}()

			fmt.Println("status: ", resp.StatusCode, resp.Status)

			d, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Read error:", err)
				return err
			}
			fmt.Println(string(d))
			return nil
		},
	}

	return cmd
}
