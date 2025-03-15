package main

import (
	"database/sql"
	"fmt"
	server "forum/src"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var err error

func init() {
	// open data base
	db, err = sql.Open("sqlite3", "./my.db")
	if err != nil {
		log.Fatal(err)
	}

	// read tables
	sqlFile, err := os.ReadFile("./src/tables.sql")
	if err != nil {
		log.Fatal("Error reading SQL file:", err)
	}

	// execute tables
	_, err = db.Exec(string(sqlFile))
	if err != nil {
		log.Fatal(err)
	}

	// send db to server file
	server.SendDB(db)
}
func main() {
	defer db.Close()
	http.HandleFunc("/", server.HomePage)
	http.HandleFunc("/register", server.HandleRegister)
	http.HandleFunc("/login", server.HandleLogin)
	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
