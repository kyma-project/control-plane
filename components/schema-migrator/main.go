package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/kyma-project/control-plane/components/schema-migrator/cleaner"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	time.Sleep(20 * time.Second)
	migrateErr := invokeMigration()
	if migrateErr != nil {
		fmt.Printf("while invoking migration: %s", migrateErr)
	}

	// continue with cleanup
	err := cleaner.Halt()

	time.Sleep(5 * time.Second)

	if err != nil || migrateErr != nil {
		fmt.Printf("error during migration: %s\n", migrateErr)
		fmt.Printf("error during cleanup: %s\n", err)
		os.Exit(-1)
	}
}

func invokeMigration() error {
	envs := []string{"DB_USER", "DB_HOST", "DB_NAME", "DB_PORT",
		"DB_PASSWORD", "MIGRATION_PATH", "DIRECTION"}

	for _, env := range envs {
		_, present := os.LookupEnv(env)
		if !present {
			return fmt.Errorf("ERROR: %s is not set", env)
		}
	}

	direction := os.Getenv("DIRECTION")
	switch direction {
	case "up":
		fmt.Println("Migration UP")
	case "down":
		fmt.Println("Migration DOWN")
	default:
		return errors.New("ERROR: DIRECTION variable accepts only two values: up or down")
	}

	dbName := os.Getenv("DB_NAME")

	_, present := os.LookupEnv("DB_SSL")
	if present {
		dbName = fmt.Sprintf("%s?sslmode=%s", dbName, os.Getenv("DB_SSL"))
	}

	hostPort := net.JoinHostPort(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"))

	connectionString := fmt.Sprintf(
		"postgres://%s:%s@%s/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		hostPort,
		dbName,
	)

	fmt.Println("# WAITING FOR CONNECTION WITH DATABASE #")
	db, err := sql.Open("postgres", connectionString)

	for i := 0; i < 30 && err != nil; i++ {
		fmt.Printf("Error while connecting to the database, %s\n", err)
		db, err = sql.Open("postgres", connectionString)
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("# COULD NOT ESTABLISH CONNECTION TO DATABASE WITH CONNECTION STRING: %s", err)
	}

	fmt.Println("# STARTING MIGRATION #")

	migrationPath := fmt.Sprintf("file:///migrate/migrations/%s", os.Getenv("MIGRATION_PATH"))

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	for i := 0; i < 30 && err != nil; i++ {
		fmt.Printf("Error during driver initialization, %s\n", err)
		driver, err = postgres.WithInstance(db, &postgres.Config{})
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("# COULD NOT CREATE DATABASE CONNECTION: %s", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("error during migration initialization: %s", err)
	}

	defer m.Close()
	m.Log = &Logger{}

	if direction == "up" {
		err = m.Up()
	} else if direction == "down" {
		err = m.Down()
	}

	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("during migration: %s", err)
	} else if err == migrate.ErrNoChange {
		fmt.Println("No Changes. Migration done.")
	}

	return nil
}

type Logger struct {
}

func (l *Logger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (l *Logger) Verbose() bool {
	return false
}
