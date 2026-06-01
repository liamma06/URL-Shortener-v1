package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {

	//PostgreSQL connection setup
	sqlUrl := os.Getenv("DATABASE_URL")
	db, err := sql.Open("pgx", sqlUrl) //only validates the struct of connection, not connection itself
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return
	}
	defer db.Close() // Ensure the database connection is closed when the main function exits

	err = db.Ping() //actually tries to connect to the database server
	if err != nil {
		log.Fatalf("Cannot connect to the database server: %v", err)
	}

	http.HandleFunc("/shorten", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		fmt.Fprintf(w, "URL shortened successfully!")
	})

	http.HandleFunc("/{code}", func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")
		fmt.Fprintf(w, "Redirecting to original URL for code: %s", code)
	})

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
		return
	}
}
