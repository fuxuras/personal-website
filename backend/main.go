package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type ViewCount struct {
	Slug  string `json:"slug"`
	Count int    `json:"count"`
}

var (
	db *sql.DB
	// Mutex to prevent race conditions on specific slugs if we were using in-memory,
	// but SQLite handles concurrency well enough for this scale with WAL mode.
	// We'll use a simple mutex for the DB connection init just in case.
	dbMutex sync.Mutex
)

func main() {
	// Initialize Database
	var err error
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/views.db"
	}

	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatal(err)
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table if not exists
	createTableSQL := `CREATE TABLE IF NOT EXISTS views (
		"slug" TEXT NOT NULL PRIMARY KEY,
		"count" INTEGER DEFAULT 0
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	// Router
	mux := http.NewServeMux()
	mux.HandleFunc("/api/views/", handleViews)

	// Middleware
	handler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func handleViews(w http.ResponseWriter, r *http.Request) {
	// Extract slug from path /api/views/{slug}
	slug := r.URL.Path[len("/api/views/"):]
	if slug == "" {
		http.Error(w, "Missing slug", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getCount(w, slug)
	case http.MethodPost:
		incrementCount(w, slug)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getCount(w http.ResponseWriter, slug string) {
	var count int
	err := db.QueryRow("SELECT count FROM views WHERE slug = ?", slug).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, ViewCount{Slug: slug, Count: count})
}

func incrementCount(w http.ResponseWriter, slug string) {
	// Upsert: Insert or Update
	_, err := db.Exec(`
		INSERT INTO views (slug, count) VALUES (?, 1)
		ON CONFLICT(slug) DO UPDATE SET count = count + 1
	`, slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the new count to return it
	getCount(w, slug)
}

func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In production, you might want to restrict this to your domain
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
