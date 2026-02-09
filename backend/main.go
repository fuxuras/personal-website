package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ViewCount struct {
	Slug  string `json:"slug"`
	Count int    `json:"count"`
}

var db *sql.DB

func main() {
	loadEnvFile(".env")
	loadEnvFile("../.env")

	var err error
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/views.db"
	}

	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatal(err)
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createViewsTableSQL := `CREATE TABLE IF NOT EXISTS views (
		"slug" TEXT NOT NULL PRIMARY KEY,
		"count" INTEGER DEFAULT 0
	);`
	if _, err := db.Exec(createViewsTableSQL); err != nil {
		log.Fatal(err)
	}

	if err := ensureBlogViewsTable(); err != nil {
		log.Fatal(err)
	}

	if err := cleanupFeedTables(); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/views/", handleViews)

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
	slug := r.URL.Path[len("/api/views/"):]
	if slug == "" {
		http.Error(w, "Missing slug", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getCount(w, slug)
	case http.MethodPost:
		incrementCount(w, r, slug)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getCount(w http.ResponseWriter, slug string) {
	var baseCount int
	err := db.QueryRow("SELECT count FROM views WHERE slug = ?", slug).Scan(&baseCount)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var uniqueCount int
	err = db.QueryRow("SELECT COUNT(*) FROM blog_post_views WHERE slug = ?", slug).Scan(&uniqueCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, ViewCount{Slug: slug, Count: baseCount + uniqueCount})
}

func incrementCount(w http.ResponseWriter, r *http.Request, slug string) {
	token := strings.TrimSpace(r.Header.Get("X-Viewer-Token"))
	if token == "" {
		http.Error(w, "Missing viewer token", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		INSERT OR IGNORE INTO blog_post_views (slug, viewer_token, created_at)
		VALUES (?, ?, ?)
	`, slug, token, time.Now().UTC().Unix())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	getCount(w, slug)
}

func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Viewer-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ensureBlogViewsTable() error {
	createSQL := `CREATE TABLE IF NOT EXISTS blog_post_views (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"slug" TEXT NOT NULL,
		"viewer_token" TEXT NOT NULL,
		"created_at" INTEGER NOT NULL,
		UNIQUE(slug, viewer_token)
	);`

	_, err := db.Exec(createSQL)
	return err
}

func cleanupFeedTables() error {
	if _, err := db.Exec(`DROP TABLE IF EXISTS feed_post_views;`); err != nil {
		return err
	}
	if _, err := db.Exec(`DROP TABLE IF EXISTS feed_posts;`); err != nil {
		return err
	}
	return nil
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			value = strings.Trim(value, "\"'")
			if key != "" && os.Getenv(key) == "" {
				_ = os.Setenv(key, value)
			}
		}
	}
}
