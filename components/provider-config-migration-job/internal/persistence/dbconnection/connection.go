package dbconnection

import (
	"time"

	"github.com/gocraft/dbr/v2"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func InitializeDatabaseConnection(connectionString string, retryCount int) (*dbr.Connection, error) {
	connection, err := waitForDatabaseAccess(connectionString, retryCount)
	if err != nil {
		return nil, err
	}

	return connection, nil
}

func waitForDatabaseAccess(connString string, retryCount int) (*dbr.Connection, error) {
	var connection *dbr.Connection
	var err error
	for ; retryCount > 0; retryCount-- {
		connection, err = dbr.Open("postgres", connString, nil)
		if err != nil {
			return nil, errors.Wrap(err, "Invalid connection string")
		}

		err = connection.Ping()
		if err == nil {
			return connection, nil
		}
		err = connection.Close()
		if err != nil {
			log.Info("Failed to close database ...")
		}

		log.Info("Failed to access database, waiting 5 seconds to retry...")
		time.Sleep(5 * time.Second)
	}

	return nil, errors.New("timeout waiting for database access")
}
