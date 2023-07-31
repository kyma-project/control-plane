package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {

	user := os.Getenv("APP_DATABASE_USER")
	password := os.Getenv("APP_DATABASE_PASSWORD")
	host := os.Getenv("APP_DATABASE_HOST")
	port := os.Getenv("APP_DATABASE_PORT")
	dbname := os.Getenv("APP_DATABASE_NAME")

	// connection string
	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=disable", host, port, user, password)

	log.Println("Connecting to ", psqlconn)

	// open database

	var err error
	var db *sql.DB

	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", psqlconn)
		if err == nil {
			break
		}
		log.Println("Retrying connecting")
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// close database
	defer db.Close()

	// check db
	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Println("Retrying ping")
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	_, err = db.Exec(`CREATE DATABASE provisioner`)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	log.Printf("DB %v created\n", dbname)
}
