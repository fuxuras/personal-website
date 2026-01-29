package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ViewCount struct {
	Slug  string `json:"slug"`
	Count int    `json:"count"`
}

type FeedPost struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type FeedListResponse struct {
	Posts      []FeedPost `json:"posts"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	NextOffset int        `json:"nextOffset"`
	HasMore    bool       `json:"hasMore"`
}

var (
	db           *sql.DB
	feedUsername string
	feedPassword string
)

func main() {
	loadEnvFile(".env")
	loadEnvFile("../.env")

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

	// Create tables if not exist
	createViewsTableSQL := `CREATE TABLE IF NOT EXISTS views (
		"slug" TEXT NOT NULL PRIMARY KEY,
		"count" INTEGER DEFAULT 0
	);`
	_, err = db.Exec(createViewsTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	createFeedTableSQL := `CREATE TABLE IF NOT EXISTS feed_posts (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"title" TEXT NOT NULL,
		"content" TEXT NOT NULL,
		"created_at" INTEGER NOT NULL
	);`
	_, err = db.Exec(createFeedTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	feedUsername = os.Getenv("FEED_USERNAME")
	feedPassword = os.Getenv("FEED_PASSWORD")
	if feedUsername == "" || feedPassword == "" {
		log.Println("Warning: FEED_USERNAME or FEED_PASSWORD is not set; feed creation will be disabled.")
	}

	// Router
	mux := http.NewServeMux()
	mux.HandleFunc("/api/views/", handleViews)
	mux.HandleFunc("/api/feed", handleFeed)

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

func handleFeed(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listFeed(w, r)
	case http.MethodPost:
		createFeed(w, r)
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

func listFeed(w http.ResponseWriter, r *http.Request) {
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)
	offset := parsePositiveInt(r.URL.Query().Get("offset"), 0)

	rows, err := db.Query(`
		SELECT id, title, content, created_at
		FROM feed_posts
		ORDER BY created_at DESC, id DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	posts := make([]FeedPost, 0)
	for rows.Next() {
		var post FeedPost
		var createdAt int64
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &createdAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		post.CreatedAt = time.Unix(createdAt, 0).UTC().Format(time.RFC3339)
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nextOffset := offset + len(posts)
	hasMore := len(posts) == limit
	respondJSON(w, FeedListResponse{
		Posts:      posts,
		Offset:     offset,
		Limit:      limit,
		NextOffset: nextOffset,
		HasMore:    hasMore,
	})
}

func createFeed(w http.ResponseWriter, r *http.Request) {
	if !requireFeedAuth(w, r) {
		return
	}

	var title string
	var content string
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var payload struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		title = strings.TrimSpace(payload.Title)
		content = strings.TrimSpace(payload.Content)
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form body", http.StatusBadRequest)
			return
		}
		title = strings.TrimSpace(r.FormValue("title"))
		content = strings.TrimSpace(r.FormValue("content"))
	}

	if title == "" || content == "" {
		http.Error(w, "Title and content are required", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	result, err := db.Exec(`
		INSERT INTO feed_posts (title, content, created_at)
		VALUES (?, ?, ?)
	`, title, content, now.Unix())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	respondJSON(w, FeedPost{
		ID:        int(id),
		Title:     title,
		Content:   content,
		CreatedAt: now.Format(time.RFC3339),
	})
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func requireFeedAuth(w http.ResponseWriter, r *http.Request) bool {
	if feedUsername == "" || feedPassword == "" {
		http.Error(w, "Feed authentication is not configured", http.StatusInternalServerError)
		return false
	}

	user, pass, ok := r.BasicAuth()
	if !ok || user != feedUsername || pass != feedPassword {
		w.Header().Set("WWW-Authenticate", `Basic realm="feed"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	return true
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
