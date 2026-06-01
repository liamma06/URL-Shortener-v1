package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // Import the pgx driver for PostgreSQL
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

	// Set up HTTP handlers
	http.HandleFunc("/shorten", shortenHandler(db))
	http.HandleFunc("/{code}", getCodeHandler(db))

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
		return
	}
}

type ShortenRequest struct {
	// The text inside `json:"url"` tells Go to map the JSON key "url" to this field
	URL string `json:"url"`
}

func shortenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		//json body parsing
		var req ShortenRequest
		defer r.Body.Close() //close body after parsing
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		originalURL := req.URL
		if originalURL == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		//generate code
		code := GenerateCode()

		//store in db
		_, err = db.Exec("INSERT INTO urls (original_url, short_code) VALUES ($1, $2)", originalURL, code)
		if err != nil {
			http.Error(w, "Failed to store URL", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Shortened URL code: %s", code)
	}
}

func getCodeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")

		var originalURL string

		//query db for original URL and handle errors and write to originalURL variable
		err := db.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", code).Scan(&originalURL)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "URL not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to retrieve URL", http.StatusInternalServerError)
			}
			return
		}
		http.Redirect(w, r, originalURL, http.StatusFound) //redirect
	}
}
