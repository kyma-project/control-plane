package storage

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	DbUser            = "admin"
	DbPass            = "nimda"
	DbName            = "broker"
	DbPort            = "5432"
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

func InitTestDBContainer(t *testing.T, ctx context.Context, hostname string) (func(), Config, error) {
	_, err := isDockerTestNetworkPresent(ctx)
	if err != nil {
		return nil, Config{}, err
	}

	isAvailable, dbCfg, err := isDBContainerAvailable(hostname, mappedPort)
	if err != nil {
		return nil, Config{}, err
	} else if !isAvailable {
		t.Log("cannot connect to DB container. Creating new Postgres container...")
	} else if isAvailable {
		return func() {}, dbCfg, nil
	}

	req := testcontainers.ContainerRequest{
		Image:        "postgres:11",
		SkipReaper:   true,
		ExposedPorts: []string{fmt.Sprintf("%s", DbPort)},
		Networks:     []string{DockerUserNetwork},
		NetworkAliases: map[string][]string{
			DockerUserNetwork: {hostname},
		},
		Env: map[string]string{
			"POSTGRES_USER":     DbUser,
			"POSTGRES_PASSWORD": DbPass,
			"POSTGRES_DB":       DbName,
		},
		WaitingFor: wait.ForListeningPort(DbPort),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		t.Logf("Failed to create contianer: %s", err.Error())
		return nil, Config{}, err
	}

	port, err := postgresContainer.MappedPort(ctx, DbPort)
	if err != nil {
		t.Logf("Failed to get mapped port for container %s : %s", postgresContainer.GetContainerID(), err.Error())
		errTerminate := postgresContainer.Terminate(ctx)
		if errTerminate != nil {
			t.Logf("Failed to terminate container %s after failing of getting mapped port: %s", postgresContainer.GetContainerID(), err.Error())
		}
		return nil, Config{}, err
	}

	cleanupFunc := func() {
		err := postgresContainer.Terminate(ctx)
		assert.NoError(t, err)
		time.Sleep(1 * time.Second)
	}

	dbCfg = makeConnectionString(hostname, port.Port())
	mappedPort = port.Port()

	return cleanupFunc, dbCfg, nil
}

func SetupTestDBContainer(ctx context.Context, hostname string) (cleanupFunc func(), dbCfg Config, err error) {
	_, err = isDockerTestNetworkPresent(ctx)
	if err != nil {
		return nil, Config{}, err
	}

	isAvailable, dbCfg, err := isDBContainerAvailable(hostname, mappedPort)
	if err != nil {
		return nil, Config{}, err
	} else if !isAvailable {
		log.Print("cannot connect to DB container. Creating new Postgres container...")
	} else if isAvailable {
		return func() {}, dbCfg, nil
	}

	req := testcontainers.ContainerRequest{
		Image:        "postgres:11",
		SkipReaper:   true,
		ExposedPorts: []string{fmt.Sprintf("%s", DbPort)},
		Networks:     []string{DockerUserNetwork},
		NetworkAliases: map[string][]string{
			DockerUserNetwork: {hostname},
		},
		Env: map[string]string{
			"POSTGRES_USER":     DbUser,
			"POSTGRES_PASSWORD": DbPass,
			"POSTGRES_DB":       DbName,
		},
		WaitingFor: wait.ForListeningPort(DbPort),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		errors.Wrap(err, "failed to create DB contianer")
		return nil, Config{}, err
	}

	port, err := postgresContainer.MappedPort(ctx, DbPort)
	if err != nil {
		log.Printf("Failed to get mapped port for container %s : %s", postgresContainer.GetContainerID(), err)
		errTerminate := postgresContainer.Terminate(ctx)
		if errTerminate != nil {
			log.Printf("Failed to terminate container %s after failing of getting mapped port: %s", postgresContainer.GetContainerID(), err)
		}
		return nil, Config{}, err
	}

	cleanupFunc = func() {
		err = postgresContainer.Terminate(ctx)
		if err != nil {
			errors.Wrap(err, "failed to remove docker DB container...")
		}
		time.Sleep(1 * time.Second)
	}

	dbCfg = makeConnectionString(hostname, port.Port())
	mappedPort = port.Port()

	return cleanupFunc, dbCfg, err
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
			log.Printf("File %s applied to database", file.Name())
		}
	}

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
			log.Printf("File %s applied to database", file.Name())
		}
	}

	return cleanupFunc, nil
}

func isDockerTestNetworkPresent(ctx context.Context) (bool, error) {
	netReq := testcontainers.NetworkRequest{
		Name:   DockerUserNetwork,
		Driver: "bridge",
	}
	provider, err := testcontainers.NewDockerProvider()

	if err != nil || provider == nil {
		return false, errors.Wrap(err, "Failed to use Docker provider to access network information")
	}

	_, err = provider.GetNetwork(ctx, netReq)

	if err == nil {
		return true, nil
	}

	return false, nil
}

func createTestNetworkForDB(ctx context.Context) (testcontainers.Network, error) {
	netReq := testcontainers.NetworkRequest{
		Name:   DockerUserNetwork,
		Driver: "bridge",
	}
	provider, err := testcontainers.NewDockerProvider()

	if err != nil || provider == nil {
		return nil, errors.Wrap(err, "Failed to use Docker provider to access network information")
	}

	createdNetwork, err := provider.CreateNetwork(ctx, netReq)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to create docker user network")
	}

	return createdNetwork, nil
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

	cleanupFunc := func() {
		err = createdNetwork.Remove(ctx)
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

	cleanupFunc = func() {
		err = createdNetwork.Remove(ctx)
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
