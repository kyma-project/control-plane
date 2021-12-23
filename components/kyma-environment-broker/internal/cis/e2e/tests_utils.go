package e2e

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	globalAccountID = "48bc1348-66db-44a4-8a17-051458bc7365"
)

func initTestDBInstancesTables(t *testing.T, connectionURL string) error {
	connection, err := postsql.WaitForDatabaseAccess(connectionURL, 10, 100*time.Millisecond, logrus.New())
	if err != nil {
		t.Logf("Cannot connect to database with URL %s", connectionURL)
		return err
	}

	dirPath := "./../../../../schema-migrator/migrations/kyma-environment-broker/"
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		t.Logf("Cannot read files from directory %s", dirPath)
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "up.sql") {
			v, err := ioutil.ReadFile(dirPath + file.Name())
			if err != nil {
				t.Logf("Cannot read file %s", file.Name())
			}
			if _, err = connection.Exec(string(v)); err != nil {
				t.Logf("Cannot apply file %s", file.Name())
				return err
			}
			t.Logf("File %s applied to database", file.Name())
		}
	}

	return nil
}

func fixInstances() []internal.Instance {
	var instances []internal.Instance

	for index, subAccountID := range []string{
		"4dd1551a-6dc8-4c4b-a0e2-3d3913cb7a31",
		"f6a7453c-a86d-4d83-a657-dbd65a7fcde1",
		"1a7b635d-baf6-4bbb-9573-bf21babc914d",
		"9d734189-e980-4109-b6f7-cacdb16fc44a",
		"25f4a3c3-d200-4cc8-b051-d46c540a381f",
		"efd1c2ee-6b59-430a-8b73-f180a33cdc94",
		"75410b1d-573d-46dd-9a2c-cd8e5fc18822",
		"36657e59-3f8f-4867-8d90-f4d450a500df",
		"fb5bcbc4-4d68-41f0-9f64-afda61f28cd1",
		"3e728728-bca1-480a-af7a-c8b4109e222d",
		"451245fc-78bc-457d-a57a-c574dcc1e7cd",
		"c21c94fc-c285-42b1-aef0-8e1f9b6879e5",
		"723abcdf-07e2-4d46-b75d-31ca43c1c030",
		"e1584649-df83-462a-b893-607e06f6f730",
		"65af4a97-94f3-45de-9617-a20c10399641",
		"1e786ce7-6566-40d7-9859-1d3bef146957",
		"20c55392-dffa-4005-996a-1fc986cb1205",
		"224b0e41-f3ae-49b7-9b1c-dc337db13584",
		"8377c743-984b-4fee-ae4f-519140e160ea",
		"08215ad3-386a-47e2-b77b-7e5f4e2a5c76",
		"c23479b8-b0a2-4685-81f5-df0cd3de7e3a",
		"83be2d80-aae5-4694-a09d-a5b1de335789",
		"fcd36951-c0ea-4047-8023-42beaecd41cc",
		"c4de62bb-6d0c-475d-a209-db998ffd1f02",
		"a052e2c5-b7f5-4bb0-aeff-e737f09fd221",
		"5235ce2a-170b-43b3-a02a-3c1574e4a167",
		"c5b584b0-0392-4673-bb97-a4553f9b1018",
		"ab8954a2-7b83-4998-8b50-c96402cabe9e",
		"885c8cdd-f69c-4700-935b-7b8ef0fda965",
		"02bdc38a-b276-4f9e-9cb7-739032bd897d",
		"b3422a0f-c90a-4fef-b420-29a2e4817bf1",
		"c88ca349-fa08-48b7-9ef6-897660e4c4f9",
		"83067aa2-d43c-4141-8e07-eae5b6a30966",
		"5a634683-21a2-4fdd-9514-03144006eee4",
		"369d0081-9330-4870-a4fd-932436135a38",
		"c8464176-28a4-4c82-a147-384b127a96cd",
		"b6695987-0eb3-4445-837e-e8c8dc700661",
		"74cea7ff-8ca5-4106-92eb-d0ef9af21e51",
		"98ea0567-89e1-4287-9e0b-31a049813cea",
		"730e1047-ba9d-482c-afea-4fac2a4a933b",
	} {
		instance := internal.Instance{
			InstanceID:      fmt.Sprintf("instance-%d", index),
			RuntimeID:       uuid.New().String(),
			SubAccountID:    subAccountID,
			GlobalAccountID: globalAccountID,
		}

		instances = append(instances, instance)
	}

	return instances
}
