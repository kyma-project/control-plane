package command

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewMigrationAllCommand() *cobra.Command {
	cmd := &MigrationAllCommand{}

	cobraCmd := &cobra.Command{
		Use:   `migrate-all`,
		Short: `Triggers full SC migration accepting json objects as input.`,
		Long:  `Migrates all instances that are feed through stdin in the form of json objects`,
		Example: `  ers migrate -w2 -d	Triggers migration starting two workers`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			// for possible param validation
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd.log = logrus.New()
			cmd.log.Out = os.Stdout

			return cmd.Run()
		},
	}

	cobraCmd.Flags().IntVarP(&cmd.workers, "workers", "w", 2, "Number of workers for processing instances.")
	cobraCmd.Flags().IntVarP(&cmd.buffer, "buffer", "b", 10, "Size of buffer for processed instances.")
	cobraCmd.Flags().Int64VarP(&cmd.recheck, "recheck", "r", 10, "Time after 'in progress' instances should be rechecked again in seconds.")

	cmd.corbaCmd = cobraCmd

	return cobraCmd
}

type MigrationAllCommand struct {
	corbaCmd  *cobra.Command
	source    string
	workers   int
	buffer    int
	recheck   int64
	wg        sync.WaitGroup
	log       *logrus.Logger
	ersClient client.Client
}

func (c *MigrationAllCommand) Run() error {
	ersClient, err := client.NewErsClient(ers.GlobalOpts.ErsUrl())
	c.ersClient = ersClient
	if err != nil {
		return fmt.Errorf("while initializing ers client: %w", err)
	}
	defer ersClient.Close()

	c.log.Infof("Creating a migrator with %d workers", c.workers)
	payloads := make(chan ers.Work, c.buffer)

	for w := 0; w < c.workers; w++ {
		go c.worker(w, payloads)
	}

	reader := bufio.NewReader(os.Stdin)
	dec := json.NewDecoder(reader)

	for dec.More() {
		var instance ers.Instance
		err := dec.Decode(&instance)
		if err != nil {
			c.log.Fatal(err)
		}

		c.log.Debugf("Read: %s\n", instance)
		c.log.Infof("Passing instance %s to workers", instance.Name)

		payloads <- ers.Work{instance, 0}
		c.wg.Add(1)
	}

	c.wg.Wait()
	close(payloads)

	return nil
}

type Worker struct {
	color string
}

func (c *MigrationAllCommand) worker(id int, workChannel chan ers.Work) {

	for work := range workChannel {
		now := time.Now().Unix()
		delta := now - work.ProcessedTimestamp
		if delta < c.recheck {
			workChannel <- work
			continue
		}

		c.log.Infof("[Worker %d] Processing instance %s", id, work.Instance.Name)

		instance := work.Instance
		if instance.Migrated {
			c.log.Infof("[Worker %d] %sInstance %s migrated%s",
				id, Green, instance.Name, Reset)
			c.wg.Done()
		} else {
			c.log.Infof("[Worker %d] Instance %s not yet migrated",
				id, instance.Name)
			c.log.Infof("[Worker %d] Instance %s - refreshing status",
				id, instance.Name)
			refreshed, err := c.ersClient.GetOne(instance.Id)
			if err != nil {
				c.log.Errorf("Error while loading the instance %s %e", instance.Name, err)
				workChannel <- work
			}
			work.Instance = *refreshed

			if refreshed == nil {
				c.log.Errorf("[Worker %d] Instance %s is missing", id, instance.Name)
				workChannel <- work
			}

			if !refreshed.Migrated {
				c.log.Infof("[Worker %d] %sTrigerring migration %s%s", id, Green, instance.Name, Reset)
				c.log.Infof("[Worker %d] %sWorker number is %d%s", id, Red, c.workers, Reset)

				err := c.ersClient.Migrate(instance.Id)
				c.log.Infof("[Worker %d] %sMigration request sent %s%s", id, Green, instance.Name, Reset)

				if err != nil {
					c.log.Errorf("Error while loading the instance %s %e", instance.Name, err)
				} else {
					c.log.Infof("[Worker %d] %sMigration request sent %s%s", id, Green, instance.Name, Reset)
				}
			}

			c.log.Infof("[Worker %d] Checking instance %s. Processing again after %v s", id, work.Instance.Name, c.recheck)
			work.ProcessedTimestamp = now
			workChannel <- work
		}
	}
}
