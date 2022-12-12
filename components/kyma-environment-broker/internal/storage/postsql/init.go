package postsql

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gocraft/dbr"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

const (
	schemaName             = "public"
	InstancesTableName     = "instances"
	OperationTableName     = "operations"
	OrchestrationTableName = "orchestrations"
	RuntimeStateTableName  = "runtime_states"
	CreatedAtField         = "created_at"
)

// InitializeDatabase opens database connection and initializes schema if it does not exist
func InitializeDatabase(connectionURL string, retries int, log logrus.FieldLogger) (*dbr.Connection, error) {
	connection, err := WaitForDatabaseAccess(connectionURL, retries, 100*time.Millisecond, log)
	if err != nil {
		return nil, err
	}

	initialized, err := CheckIfDatabaseInitialized(connection)
	if err != nil {
		closeDBConnection(connection, log)
		return nil, fmt.Errorf("failed to check if database is initialized: %w", err)
	}
	if initialized {
		log.Info("Database already initialized")
		return connection, nil
	}

	return connection, nil
}

func closeDBConnection(db *dbr.Connection, log logrus.FieldLogger) {
	err := db.Close()
	if err != nil {
		log.Warnf("Failed to close database connection: %s", err.Error())
	}
}

const TableNotExistsError = "42P01"

func CheckIfDatabaseInitialized(db *dbr.Connection) (bool, error) {
	checkQuery := fmt.Sprintf(`SELECT '%s.%s'::regclass;`, schemaName, InstancesTableName)

	row := db.QueryRow(checkQuery)

	var tableName string
	err := row.Scan(&tableName)

	if err != nil {
		psqlErr, converted := err.(*pq.Error)

		if converted && psqlErr.Code == TableNotExistsError {
			return false, nil
		}

		return false, fmt.Errorf("failed to check if database is initialized: %w", err)
	}

	return tableName == InstancesTableName, nil
}

func WaitForDatabaseAccess(connString string, retryCount int, sleepTime time.Duration, log logrus.FieldLogger) (*dbr.Connection, error) {
	var connection *dbr.Connection
	var err error

	re := regexp.MustCompile(`password=.*?\s`)
	log.Info(re.ReplaceAllString(connString, ""))

	for ; retryCount > 0; retryCount-- {
		connection, err = dbr.Open("postgres", connString, nil)
		if err != nil {
			return nil, fmt.Errorf("invalid connection string: %w", err)
		}

		err = connection.Ping()
		if err == nil {
			return connection, nil
		}
		log.Warnf("Database Connection failed: %s", err.Error())

		err = connection.Close()
		if err != nil {
			log.Info("Failed to close database ...")
		}

		log.Infof("Failed to access database, waiting %v to retry...", sleepTime)
		time.Sleep(sleepTime)
	}

	return nil, fmt.Errorf("timeout waiting for database access")
}
