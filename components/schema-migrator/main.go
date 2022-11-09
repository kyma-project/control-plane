package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/kyma-project/control-plane/components/schema-migrator/cleaner"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	time.Sleep(20 * time.Second)
	migrateErr := invokeMigration()
	if migrateErr != nil {
		log.Printf("while invoking migration: %s", migrateErr)
	}

	// continue with cleanup
	err := cleaner.Halt()

	time.Sleep(5 * time.Second)

	if err != nil || migrateErr != nil {
		log.Printf("error during migration: %s\n", migrateErr)
		log.Printf("error during cleanup: %s\n", err)
		os.Exit(-1)
	}
}

func invokeMigration() error {
	envs := []string{
		"DB_USER", "DB_HOST", "DB_NAME", "DB_PORT",
		"DB_PASSWORD", "MIGRATION_PATH", "DIRECTION",
	}

	for _, env := range envs {
		_, present := os.LookupEnv(env)
		if !present {
			return fmt.Errorf("ERROR: %s is not set", env)
		}
	}

	direction := os.Getenv("DIRECTION")
	switch direction {
	case "up":
		log.Println("Migration UP")
	case "down":
		log.Println("Migration DOWN")
	default:
		return errors.New("ERROR: DIRECTION variable accepts only two values: up or down")
	}

	dbName := os.Getenv("DB_NAME")

	_, present := os.LookupEnv("DB_SSL")
	if present {
		dbName = fmt.Sprintf("%s?sslmode=%s", dbName, os.Getenv("DB_SSL"))

		_, present := os.LookupEnv("DB_SSLROOTCERT")
		if present {
			dbName = fmt.Sprintf("%s&sslrootcert=%s", dbName, os.Getenv("DB_SSLROOTCERT"))
		}
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

	log.Println("# WAITING FOR CONNECTION WITH DATABASE #")
	db, err := sql.Open("postgres", connectionString)

	for i := 0; i < 30 && err != nil; i++ {
		fmt.Printf("Error while connecting to the database, %s\n", err)
		db, err = sql.Open("postgres", connectionString)
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("# COULD NOT ESTABLISH CONNECTION TO DATABASE WITH CONNECTION STRING: %w", err)
	}

	log.Println("# STARTING COPY MIGRATION FILES #")
	migrationTempPath := fmt.Sprintf("tmp-migrations-%s-*", os.Getenv("MIGRATION_PATH"))

	tmpDir, err := os.MkdirTemp("/migrate", migrationTempPath)
	if err != nil {
		return fmt.Errorf("# COULD NOT CREATE TEMPORARY FILE FOR MIGRATION: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	migrationNewPath := fmt.Sprintf("new-migrations/%s", os.Getenv("MIGRATION_PATH"))
	err = copyDir(migrationNewPath, tmpDir)
	if err != nil {
		log.Printf("# COULD NOT COPY NEW MIGRATION FILES: %s\n", err)
	}

	migrationOldPath := fmt.Sprintf("migrations/%s", os.Getenv("MIGRATION_PATH"))
	err = copyDir(migrationOldPath, tmpDir)
	if err != nil {
		return fmt.Errorf("# COULD NOT COPY OLD MIGRATION FILES: %w", err)
	}

	log.Println("# STARTING MIGRATION #")

	migrationPath := fmt.Sprintf("file:///%s", tmpDir)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	for i := 0; i < 30 && err != nil; i++ {
		fmt.Printf("Error during driver initialization, %s\n", err)
		driver, err = postgres.WithInstance(db, &postgres.Config{})
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("# COULD NOT CREATE DATABASE CONNECTION: %w", err)
	}

	migrateInstance, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("error during migration initialization: %w", err)
	}

	defer func(migrateInstance *migrate.Migrate) {
		err, _ := migrateInstance.Close()
		if err != nil {
			log.Printf("error during migrate instance close: %s\n", err)
		}
	}(migrateInstance)
	migrateInstance.Log = &Logger{}

	if direction == "up" {
		err = migrateInstance.Up()
	} else if direction == "down" {
		err = migrateInstance.Down()
	}

	if err != nil && !errors.Is(migrate.ErrNoChange, err) {
		return fmt.Errorf("during migration: %w", err)
	} else if errors.Is(migrate.ErrNoChange, err) {
		log.Println("No Changes. Migration done.")
	}

	return nil
}

type Logger struct{}

func (l *Logger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (l *Logger) Verbose() bool {
	return false
}

func copyFile(src, dst string) error {
	rd, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer rd.Close()

	wr, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer wr.Close()

	_, err = io.Copy(wr, rd)
	if err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("retrieving fileinfo: %w", err)
	}

	return os.Chmod(dst, srcInfo.Mode())
}

func copyDir(src string, dst string) error {
	files, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error during reading directory content: %w", err)
	}

	for _, file := range files {
		srcFile := path.Join(src, file.Name())
		dstFile := path.Join(dst, file.Name())
		fileExt := filepath.Ext(srcFile)
		if fileExt == ".sql" {
			err = copyFile(srcFile, dstFile)
			if err != nil {
				return fmt.Errorf("error during: %w", err)
			}
		}
	}

	return nil
}
