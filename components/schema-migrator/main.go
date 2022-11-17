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

	var fs FileSystem = osFS{}
	migrationTempPath := fmt.Sprintf("tmp-migrations-%s-*", os.Getenv("MIGRATION_PATH"))

	tmpDir, err := os.MkdirTemp("/migrate", migrationTempPath)
	if err != nil {
		return fmt.Errorf("# COULD NOT CREATE TEMPORARY FILE FOR MIGRATION: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	migrationNewPath := fmt.Sprintf("new-migrations/%s", os.Getenv("MIGRATION_PATH"))
	err = copyDir(migrationNewPath, tmpDir, fs)
	if err != nil {
		log.Printf("# COULD NOT COPY NEW MIGRATION FILES: %s\n", err)
	}

	migrationOldPath := fmt.Sprintf("migrations/%s", os.Getenv("MIGRATION_PATH"))
	err = copyDir(migrationOldPath, tmpDir, fs)
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

func copyFile(src, dst string, fs FileSystem) error {
	rd, err := fs.Open(src)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer rd.Close()

	wr, err := fs.Create(dst)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer wr.Close()

	_, err = fs.Copy(wr, rd)
	if err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	srcInfo, err := fs.Stat(src)
	if err != nil {
		return fmt.Errorf("retrieving fileinfo: %w", err)
	}

	return fs.Chmod(dst, srcInfo.Mode())
}

func copyDir(src, dst string, fs FileSystem) error {
	files, err := fs.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error during reading directory content: %w", err)
	}

	for _, file := range files {
		srcFile := path.Join(src, file.Name())
		dstFile := path.Join(dst, file.Name())
		fileExt := filepath.Ext(srcFile)
		if fileExt == ".sql" {
			err = copyFile(srcFile, dstFile, fs)
			if err != nil {
				return fmt.Errorf("error during: %w", err)
			}
		}
	}

	return nil
}
