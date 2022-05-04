package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Attachment struct {
	Text  string `json:"text"`
	Color string `json:"color"`
	Title string `json:"title"`
}

type SlackRequestBody struct {
	Text        string       `json:"text"`
	Icon_emoji  string       `json:"icon_emoji"`
	Attachments []Attachment `json:"attachments"`
	Username    string       `json:"username"`
}

const ICON_EMOJI = ":high_brightness:"
const SLACK_USER_NAME = "upgrade-info"
const SLACK_COLOR = "#36a64f" //green
const Gardener_Namespace_Prefix = "garden-kyma"
const KCP_Prefix = "kcp"
const PROD_Postfix = "-prod"

var upgradeOpts = []string{"parallel-workers", "schedule", "strategy",
	"target", "target-exclude", "verbose", "version"}

// SendSlackNotification will post message including attachments to slackhookUrl.
func SendSlackNotification(title string, cobraCmd *cobra.Command, output string) error {
	slackhookUrl := GlobalOpts.SlackAPIURL()
	text_msg := "New " + title + " is triggerred on " + getClusterType()
	triggeredCmd := revertUpgradeOpts(title, cobraCmd)
	attachment := Attachment{
		Color: SLACK_COLOR,
		Text:  triggeredCmd + "\n" + output,
	}

	slackBody, _ := json.Marshal(SlackRequestBody{Text: text_msg, Icon_emoji: ICON_EMOJI, Username: SLACK_USER_NAME,
		Attachments: []Attachment{attachment}})
	req, err := http.NewRequest(http.MethodPost, slackhookUrl, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	// Drain response body and close, return error to context if there isn't any.
	defer func() {
		derr := drainResponseBody(resp.Body)
		if err == nil {
			err = derr
		}
		cerr := resp.Body.Close()
		if err == nil {
			err = cerr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("calling %s returned %s status", slackhookUrl, resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("while decoding slack response body")
	}
	bodyString := string(bodyBytes)
	if bodyString != "ok" {
		return fmt.Errorf("non-ok response returned from Slack")
	}
	return nil
}

func drainResponseBody(body io.Reader) error {
	if body == nil {
		return nil
	}
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, 4096))
	return err
}

func getClusterType() string {
	clusterType := strings.Replace(GlobalOpts.GardenerNamespace(), Gardener_Namespace_Prefix, KCP_Prefix, -1)

	if clusterType == KCP_Prefix {
		clusterType = KCP_Prefix + PROD_Postfix
	}
	return clusterType
}

func revertUpgradeOpts(title string, cobraCmd *cobra.Command) string {
	commnd := "kcp " + title
	cobraCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		for _, col := range upgradeOpts {
			if flag.Name == col && flag.Value.String() != "" {
				if (flag.Name == "strategy" && flag.Value.String() == "parallel") ||
					((flag.Name == "verbose" || flag.Name == "parallel-workers") && flag.Value.String() == "0") ||
					((flag.Name == "target" || flag.Name == "target-exclude") && flag.Value.String() == "[]") {
					continue
				} else if flag.Name == "target" || flag.Name == "target-exclude" {
					commnd = commnd + " --" + flag.Name + " " + strings.Trim(flag.Value.String(), "[]")
				} else {
					commnd = commnd + " --" + flag.Name + " " + flag.Value.String()
				}
			}
		}
	})
	return commnd
}
