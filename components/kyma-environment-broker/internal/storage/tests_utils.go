package storage

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gocraft/dbr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	DbUser            = "admin"
	DbPass            = "nimda"
	DbName            = "broker"
	DbPort            = "5432/tcp"
	DockerUserNetwork = "test_network"
	EnvPipelineBuild  = "PIPELINE_BUILD"
)

var mappedPort string

func makeConnectionString(hostname string, port string) Config {
	host := "localhost"
	if os.Getenv(EnvPipelineBuild) != "" {
		host = hostname
		port = DbPort
	}

	cfg := Config{
		Host:      host,
		User:      DbUser,
		Password:  DbPass,
		Port:      port,
		Name:      DbName,
		SSLMode:   "disable",
		SecretKey: "$C&F)H@McQfTjWnZr4u7x!A%D*G-KaNd",

		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}
	return cfg
}

func CloseDatabase(t *testing.T, connection *dbr.Connection) {
	if connection != nil {
		err := connection.Close()
		assert.Nil(t, err, "Failed to close db connection")
	}
}

func closeDBConnection(connection *dbr.Connection) {
	if connection != nil {
		err := connection.Close()
		if err != nil {
			log.Printf("failed to close db connection: %v", err)
		}
	}
}

func InitTestDBContainer(log func(format string, args ...interface{}), ctx context.Context, hostname string) (func(), Config, error) {
	_, err := isDockerTestNetworkPresent(ctx)
	if err != nil {
		return nil, Config{}, err
	}

	isAvailable, dbCfg, err := isDBContainerAvailable(hostname, mappedPort)
	if err != nil {
		return nil, Config{}, err
	} else if !isAvailable {
		log("cannot connect to DB container. Creating new Postgres container...")
	} else if isAvailable {
		return func() {}, dbCfg, nil
	}

	return createDbContainer(log, hostname)
}

func InitTestDBTables(t *testing.T, connectionURL string) (func(), error) {
	connection, err := postsql.WaitForDatabaseAccess(connectionURL, 10, 100*time.Millisecond, logrus.New())
	if err != nil {
		t.Logf("Cannot connect to database with URL %s", connectionURL)
		return nil, err
	}

	cleanupFunc := func() {
		_, err = connection.Exec(clearDBQuery())
		if err != nil {
			errors.Wrap(err, "failed to clear DB tables...")
		}
	}

	initialized, err := postsql.CheckIfDatabaseInitialized(connection)
	if err != nil {
		CloseDatabase(t, connection)
		return nil, err
	} else if initialized {
		return cleanupFunc, nil
	}

	dirPath := "./../../../../../schema-migrator/migrations/kyma-environment-broker/"
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Printf("Cannot read files from directory %s", dirPath)
		return nil, err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "up.sql") {
			v, err := ioutil.ReadFile(dirPath + file.Name())
			if err != nil {
				log.Printf("Cannot read file %s", file.Name())
			}
			if _, err = connection.Exec(string(v)); err != nil {
				log.Printf("Cannot apply file %s", file.Name())
				return nil, err
			}
		}
	}
	log.Printf("Files applied to database")

	return cleanupFunc, nil
}

func SetupTestDBTables(connectionURL string) (cleanupFunc func(), err error) {
	connection, err := postsql.WaitForDatabaseAccess(connectionURL, 10, 100*time.Millisecond, logrus.New())
	if err != nil {
		log.Printf("Cannot connect to database with URL %s", connectionURL)
		return nil, err
	}

	cleanupFunc = func() {
		_, err = connection.Exec(clearDBQuery())
		if err != nil {
			errors.Wrap(err, "failed to clear DB tables...")
		}
	}

	initialized, err := postsql.CheckIfDatabaseInitialized(connection)
	if err != nil {
		closeDBConnection(connection)
		return nil, err
	} else if initialized {
		return cleanupFunc, nil
	}

	dirPath := "./../../../../../schema-migrator/migrations/kyma-environment-broker/"
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Printf("Cannot read files from directory %s", dirPath)
		return nil, err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "up.sql") {
			v, err := ioutil.ReadFile(dirPath + file.Name())
			if err != nil {
				log.Printf("Cannot read file %s", file.Name())
			}
			if _, err = connection.Exec(string(v)); err != nil {
				log.Printf("Cannot apply file %s", file.Name())
				return nil, err
			}
		}
	}
	log.Printf("Files applied to database")

	return cleanupFunc, nil
}

func isDockerTestNetworkPresent(ctx context.Context) (bool, error) {

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	filters := filters.NewArgs()
	filters.Add("name", DockerUserNetwork)
	filters.Add("driver", "bridge")
	list, err := cli.NetworkList(context.Background(), types.NetworkListOptions{Filters: filters})

	if err == nil {
		return len(list) == 1, nil
	}

	return false, nil
}

func createTestNetworkForDB(ctx context.Context) (*types.NetworkResource, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a Docker client")
	}

	createdNetworkResponse, err := cli.NetworkCreate(context.Background(), DockerUserNetwork, types.NetworkCreate{Driver: "bridge"})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create docker user network")
	}

	filters := filters.NewArgs()
	filters.Add("id", createdNetworkResponse.ID)
	list, err := cli.NetworkList(context.Background(), types.NetworkListOptions{Filters: filters})

	if err != nil || len(list) != 1 {
		return nil, errors.Wrap(err, "network not found or not created")
	}

	return &list[0], nil
}

func EnsureTestNetworkForDB(t *testing.T, ctx context.Context) (func(), error) {
	exec.Command("systemctl start docker.service")

	networkPresent, err := isDockerTestNetworkPresent(ctx)
	if networkPresent && err == nil {
		return func() {}, nil
	}

	if os.Getenv(EnvPipelineBuild) != "" {
		return func() {}, errors.Errorf("Docker network %s does not exist", DockerUserNetwork)
	}

	createdNetwork, err := createTestNetworkForDB(ctx)

	if err != nil {
		return func() {}, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a Docker client")
	}

	cleanupFunc := func() {
		err = cli.NetworkRemove(ctx, createdNetwork.ID)
		assert.NoError(t, err)
		time.Sleep(1 * time.Second)
	}

	return cleanupFunc, nil
}

func SetupTestNetworkForDB(ctx context.Context) (cleanupFunc func(), err error) {
	exec.Command("systemctl start docker.service")

	networkPresent, err := isDockerTestNetworkPresent(ctx)
	if networkPresent && err == nil {
		return func() {}, nil
	}

	if os.Getenv(EnvPipelineBuild) != "" {
		return func() {}, errors.Errorf("Docker network %s does not exist", DockerUserNetwork)
	}

	createdNetwork, err := createTestNetworkForDB(ctx)

	if err != nil {
		return func() {}, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a Docker client")
	}
	cleanupFunc = func() {
		err = cli.NetworkRemove(ctx, createdNetwork.ID)
		if err != nil {
			errors.Wrap(err, "failed to remove docker network:"+DockerUserNetwork)
		}
		time.Sleep(1 * time.Second)
	}

	return cleanupFunc, err
}

func isDBContainerAvailable(hostname, port string) (isAvailable bool, dbCfg Config, err error) {
	dbCfg = makeConnectionString(hostname, port)

	connection, err := dbr.Open("postgres", dbCfg.ConnectionURL(), nil)
	if err != nil {
		return false, Config{}, errors.Wrap(err, "invalid connection string")
	}

	defer func(c *dbr.Connection) {
		err = c.Close()
		if err != nil {
			errors.Wrap(err, "failed to close database connection...")
		}
	}(connection)

	err = connection.Ping()
	if err == nil {
		return true, dbCfg, nil
	}

	return false, Config{}, err
}

func clearDBQuery() string {
	return fmt.Sprintf("TRUNCATE TABLE %s, %s, %s, %s RESTART IDENTITY CASCADE",
		postsql.InstancesTableName,
		postsql.OperationTableName,
		postsql.OrchestrationTableName,
		postsql.RuntimeStateTableName,
	)
}

func createDbContainer(log func(format string, args ...interface{}), hostname string) (func(), Config, error) {

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	body, err := cli.ContainerCreate(context.Background(),
		&container.Config{
			Image: "postgres:11",
			ExposedPorts: nat.PortSet{
				DbPort: struct{}{},
			},
			Env: []string{
				fmt.Sprintf("POSTGRES_USER=%s", DbUser),
				fmt.Sprintf("POSTGRES_PASSWORD=%s", DbPass),
				fmt.Sprintf("POSTGRES_DB=%s", DbName),
			},
		},
		&container.HostConfig{
			NetworkMode:     DockerUserNetwork,
			PublishAllPorts: true,
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				DockerUserNetwork: {
					Aliases: []string{
						hostname,
					},
				},
			},
		},
		&v1.Platform{},
		"")

	if err := cli.ContainerStart(context.Background(), body.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	err = waitFor(cli, body.ID, "database system is ready to accept connections")
	if err != nil {
		log("Failed to query container's configs: %s", err)
		return nil, Config{}, err
	}

	filters := filters.NewArgs()
	filters.Add("ID", body.ID)
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})

	if err != nil || len(containers) == 0 {
		log("no containers found: %s", err)
		return nil, Config{}, err
	}

	var container *types.Container
	for _, cont := range containers {
		if cont.ID == body.ID {
			container = &cont
			break
		}
	}

	if container == nil {
		log("no container found: %s", err)
		return nil, Config{}, err
	}

	ports := container.Ports
	if len(ports) != 1 {
		log("more or less then one binding found")
		return nil, Config{}, errors.New("more or less then one binding found")
	}

	cleanupFunc := func() {
		cli.ContainerRemove(context.Background(), body.ID, types.ContainerRemoveOptions{true, true, true})
	}

	dbCfg := makeConnectionString(hostname, fmt.Sprint(ports[0].PublicPort))

	return cleanupFunc, dbCfg, nil
}

func waitFor(cli *client.Client, containerId string, text string) error {
	return wait.PollImmediate(3*time.Second, 10*time.Second, func() (done bool, err error) {
		out, err := cli.ContainerLogs(context.Background(), containerId, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			panic(err)
		}

		bufReader := bufio.NewReader(out)
		defer out.Close()

		for line, isPrefix, err := bufReader.ReadLine(); err == nil; line, isPrefix, err = bufReader.ReadLine() {
			if !isPrefix && strings.Contains(string(line), text) {
				return true, nil
			}
		}

		return false, err
	})
}
