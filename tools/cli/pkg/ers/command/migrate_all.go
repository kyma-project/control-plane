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
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/spf13/cobra"
)

func NewMigrationAllCommand(log logger.Logger) *cobra.Command {
	cmd := &MigrationAllCommand{}

	cobraCmd := &cobra.Command{
		Use:   `migrate-all`,
		Short: `Triggers full SC migration accepting json objects as input.`,
		Long:  `Migrates all instances that are feed through stdin in the form of json objects`,
		Example: `  ers migrate -w2 -d	Triggers migration starting two workers`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			// for possible param validation
			cmd.log = logger.New()
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		},
	}

	cobraCmd.Flags().IntVarP(&cmd.workers, "workers", "w", 2, "Number of workers for processing instances.")
	cobraCmd.Flags().IntVarP(&cmd.buffer, "buffer", "b", 10, "Size of buffer for processed instances.")
	cobraCmd.Flags().Int64VarP(&cmd.recheck, "recheck", "r", 10, "Time after 'in progress' instances should be rechecked again in seconds.")

	cobraCmd.Flags().BoolVarP(&cmd.dryRun, "mock-ers", "", true, "Use fake ERS client to test")

	cmd.corbaCmd = cobraCmd
	cmd.stats = NewStats()

	return cobraCmd
}

type MigrationAllCommand struct {
	corbaCmd  *cobra.Command
	source    string
	workers   int
	buffer    int
	recheck   int64
	wg        sync.WaitGroup
	log       logger.Logger
	ersClient client.Client
	dryRun    bool
	stats     *Stats
}

func (c *MigrationAllCommand) Run() error {
	c.ersClient = client.NewFake()
	if !c.dryRun {
		ersClient, err := client.NewErsClient()
		c.ersClient = ersClient
		if err != nil {
			return fmt.Errorf("while initializing ers client: %w", err)
		}
		defer ersClient.Close()
	}

	c.log.Infof("Creating a migrator with %d workers", c.workers)
	payloads := make(chan ers.Work, c.buffer)

	for w := 0; w < c.workers; w++ {
		go c.simpleWorker(w, payloads)
	}

	c.log.Debugf("Preparing readers")
	reader := bufio.NewReader(os.Stdin)
	dec := json.NewDecoder(reader)

	var instances []ers.Instance
	err := dec.Decode(&instances)
	if err != nil {
		c.log.Errorf("Unable to decode input: %s", err.Error())
		return err
	}

	for _, instance := range instances {

		c.log.Debugf("Read: %s\n", instance)
		c.log.Infof("Passing instance %s to workers", instance.Name)

		payloads <- ers.Work{instance, 0}
		c.wg.Add(1)
		c.stats.Add()
	}

	c.wg.Wait()
	close(payloads)
	c.stats.Print()

	c.log.Debugf("Closing...")
	return nil
}

type Worker struct {
	color string
}

func (c *MigrationAllCommand) simpleWorker(workerId int, workChannel chan ers.Work) {
	for work := range workChannel {
		c.stats.PrintProgress()
		start := time.Now()

		c.log.Infof("[Worker %d] Processing instance %s", workerId, work.Instance.Name)
		instance := work.Instance
		if instance.Migrated {
			c.log.Infof("[Worker %d] %sInstance %s migrated%s",
				workerId, Green, instance.Name, Reset)
			c.wg.Done()
			c.stats.Done()
			continue
		}
		refreshed, err := c.ersClient.GetOne(instance.Id)
		if err != nil {
			c.log.Warnf("[Worker %d] GetOne error: %s", workerId, err.Error())
			c.wg.Done()
			c.stats.Err(instance.Id, err)
			// TODO: add retries

			continue
		}
		if refreshed.Migrated {
			c.log.Infof("[Worker %d] Refreshed %sInstance %s migrated%s",
				workerId, Green, instance.Name, Reset)
			c.wg.Done()
			c.stats.Done()
			continue
		}
		c.log.Infof("[Worker %d] Triggering migration (instanceID=%s)", workerId, instance.Id)
		c.ersClient.Migrate(instance.Id)

		c.log.Infof("[Worker %d] Instance %s not yet migrated",
			workerId, instance.Name)
		c.log.Infof("[Worker %d] Instance %s - refreshing status",
			workerId, instance.Name)

		for time.Since(start) < 20*time.Minute {
			refreshed, err := c.ersClient.GetOne(instance.Id)
			if err != nil {
				c.log.Warnf("[Worker %d] GetOne error: %s", workerId, err.Error())
			}
			if refreshed.Migrated {
				c.log.Infof("[Worker %d] Migrated: %s", workerId, instance.Id)
				break
			}
			time.Sleep(10 * time.Second)
		}
		// TODO: Add retries
		c.wg.Done()
		c.stats.Done()
	}
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
