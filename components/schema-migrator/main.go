package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
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

const connRetries = 30

//go:generate mockery --name=FileSystem
type FileSystem interface {
	Open(name string) (*os.File, error)
	Stat(name string) (os.FileInfo, error)
	Create(name string) (*os.File, error)
	Chmod(name string, mode os.FileMode) error
	Copy(dst io.Writer, src io.Reader) (int64, error)
	ReadDir(name string) ([]fs.DirEntry, error)
}

//go:generate mockery --name=MyFileInfo
type MyFileInfo interface {
	Name() string       // base name of the file
	Size() int64        // length in bytes for regular files; system-dependent for others
	Mode() os.FileMode  // file mode bits
	ModTime() time.Time // modification time
	IsDir() bool        // abbreviation for Mode().IsDir()
	Sys() any           // underlying data source (can return nil)
}

type osFS struct{}

type migrationScript struct {
	fs FileSystem
}

func (osFS) Open(name string) (*os.File, error) {
	return os.Open(name)
}
func (osFS) Create(name string) (*os.File, error) {
	return os.Create(name)
}
func (osFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
func (osFS) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}
func (osFS) Copy(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}
func (osFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	migrateErr := invokeMigration()
	if migrateErr != nil {
		log.Printf("while invoking migration: %s", migrateErr)
	}

	// continue with cleanup
	err := cleaner.Halt()

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

	for i := 0; i < connRetries && err != nil; i++ {
		fmt.Printf("Error while connecting to the database, %s. Retrying step\n", err)
		db, err = sql.Open("postgres", connectionString)
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		return fmt.Errorf("# COULD NOT ESTABLISH CONNECTION TO DATABASE WITH CONNECTION STRING: %w", err)
	}
	log.Println("# CONNECTION WITH DATABASE ESTABLISHED #")
	log.Println("# STARTING TO COPY MIGRATION FILES #")

	migrationEnvPath := os.Getenv("MIGRATION_PATH")

	migrationTempPath := fmt.Sprintf("tmp-migrations-%s-*", migrationEnvPath)

	migrationExecPath, err := os.MkdirTemp("/migrate", migrationTempPath)
	if err != nil {
		return fmt.Errorf("# COULD NOT CREATE TEMPORARY DIRECTORY FOR MIGRATION: %w", err)
	}
	defer os.RemoveAll(migrationExecPath)

	ms := migrationScript{
		fs: osFS{},
	}
	newMigrationsSrc := fmt.Sprintf("new-migrations/%s", migrationEnvPath)
	log.Println("# LOADING MIGRATION FILES FROM CONFIGMAP #")
	err = ms.copyDir(newMigrationsSrc, migrationExecPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("# NO MIGRATION FILES PROVIDED BY THE CONFIGMAP, SKIPPING STEP #")
		} else {
			return fmt.Errorf("# COULD NOT COPY MIGRATION FILES PROVIDED BY THE CONFIGMAP: %w", err)
		}
	} else {
		log.Println("# LOADING MIGRATION FILES FROM CONFIGMAP DONE #")
	}

	oldMigrationsSrc := fmt.Sprintf("migrations/%s", migrationEnvPath)
	log.Println("# LOADING EMBEDDED MIGRATION FILES FROM THE SCHEMA-MIGRATOR IMAGE #")
	err = ms.copyDir(oldMigrationsSrc, migrationExecPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("# NO MIGRATION FILES EMBEDDED TO THE SCHEMA-MIGRATOR IMAGE, SKIPPING STEP #")
		} else {
			return fmt.Errorf("# COULD NOT COPY EMBEDDED MIGRATION FILES FROM THE SCHEMA-MIGRATOR IMAGE: %w", err)
		}
	} else {
		log.Println("# LOADING EMBEDDED MIGRATION FILES FROM THE SCHEMA-MIGRATOR IMAGE DONE #")
	}

	log.Println("# INITIALIZING DRIVER #")
	driver, err := postgres.WithInstance(db, &postgres.Config{})

	for i := 0; i < connRetries && err != nil; i++ {
		fmt.Printf("Error during driver initialization, %s. Retrying step\n", err)
		driver, err = postgres.WithInstance(db, &postgres.Config{})
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		return fmt.Errorf("# COULD NOT CREATE DATABASE CONNECTION: %w", err)
	}
	log.Println("# DRIVER INITIALIZED #")
	log.Println("# STARTING MIGRATION #")

	migrationPath := fmt.Sprintf("file:///%s", migrationExecPath)

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
		log.Println("# NO CHANGES DETECTED #")
	}

	log.Println("# MIGRATION DONE #")

	currentMigrationVer, _, err := migrateInstance.Version()
	if err != nil {
		return fmt.Errorf("during acquiring active migration version: %w", err)
	}

	log.Printf("# CURRENT ACTIVE MIGRATION VERSION: %d #", currentMigrationVer)
	return nil
}

type Logger struct{}

func (l *Logger) Printf(format string, v ...interface{}) {
	fmt.Printf("Executed "+format, v...)
}

func (l *Logger) Verbose() bool {
	return false
}

func (m *migrationScript) copyFile(src, dst string) error {
	rd, err := m.fs.Open(src)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer rd.Close()

	wr, err := m.fs.Create(dst)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer wr.Close()

	_, err = m.fs.Copy(wr, rd)
	if err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	srcInfo, err := m.fs.Stat(src)
	if err != nil {
		return fmt.Errorf("retrieving fileinfo: %w", err)
	}

	return m.fs.Chmod(dst, srcInfo.Mode())
}

func (m *migrationScript) copyDir(src, dst string) error {
	files, err := m.fs.ReadDir(src)
	if err != nil {
		return err
	}

	for _, file := range files {
		srcFile := path.Join(src, file.Name())
		dstFile := path.Join(dst, file.Name())
		fileExt := filepath.Ext(srcFile)
		if fileExt == ".sql" {
			err = m.copyFile(srcFile, dstFile)
			if err != nil {
				return fmt.Errorf("error during: %w", err)
			}
		}
	}

	return nil
}
