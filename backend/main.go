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

type FeedPost struct {
	ID        int    `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	Views     int    `json:"views"`
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

	if err := ensureFeedTable(); err != nil {
		log.Fatal(err)
	}
	if err := ensureFeedViewsTable(); err != nil {
		log.Fatal(err)
	}
	if err := cleanupBlogTables(); err != nil {
		log.Fatal(err)
	}

	feedUsername = os.Getenv("FEED_USERNAME")
	feedPassword = os.Getenv("FEED_PASSWORD")
	if feedUsername == "" || feedPassword == "" {
		log.Println("Warning: FEED_USERNAME or FEED_PASSWORD is not set; feed creation will be disabled.")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/feed", handleFeed)
	mux.HandleFunc("/api/feed/", handleFeedViews)

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

func listFeed(w http.ResponseWriter, r *http.Request) {
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)
	offset := parsePositiveInt(r.URL.Query().Get("offset"), 0)

	rows, err := db.Query(`
		SELECT p.id, p.content, p.created_at, COALESCE(v.view_count, 0)
		FROM feed_posts p
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS view_count
			FROM feed_post_views
			GROUP BY post_id
		) v ON v.post_id = p.id
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
		if err := rows.Scan(&post.ID, &post.Content, &createdAt, &post.Views); err != nil {
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

	var content string
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var payload struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		content = strings.TrimSpace(payload.Content)
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form body", http.StatusBadRequest)
			return
		}
		content = strings.TrimSpace(r.FormValue("content"))
	}

	if content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	result, err := db.Exec(`
		INSERT INTO feed_posts (content, created_at)
		VALUES (?, ?)
	`, content, now.Unix())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	respondJSON(w, FeedPost{
		ID:        int(id),
		Content:   content,
		CreatedAt: now.Format(time.RFC3339),
		Views:     0,
	})
}

func handleFeedViews(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/feed/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "views" {
		http.NotFound(w, r)
		return
	}

	postID, err := strconv.Atoi(parts[0])
	if err != nil || postID <= 0 {
		http.Error(w, "Invalid post id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getFeedViews(w, postID)
	case http.MethodPost:
		incrementFeedViews(w, r, postID)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getFeedViews(w http.ResponseWriter, postID int) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM feed_post_views WHERE post_id = ?
	`, postID).Scan(&count)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{
		"postId": postID,
		"count":  count,
	})
}

func incrementFeedViews(w http.ResponseWriter, r *http.Request, postID int) {
	token := strings.TrimSpace(r.Header.Get("X-Viewer-Token"))
	if token == "" {
		http.Error(w, "Missing viewer token", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		INSERT OR IGNORE INTO feed_post_views (post_id, viewer_token, created_at)
		VALUES (?, ?, ?)
	`, postID, token, time.Now().UTC().Unix())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	getFeedViews(w, postID)
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

func ensureFeedTable() error {
	createSQL := `CREATE TABLE IF NOT EXISTS feed_posts (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"content" TEXT NOT NULL,
		"created_at" INTEGER NOT NULL
	);`

	_, err := db.Exec(createSQL)
	return err
}

func ensureFeedViewsTable() error {
	createSQL := `CREATE TABLE IF NOT EXISTS feed_post_views (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"post_id" INTEGER NOT NULL,
		"viewer_token" TEXT NOT NULL,
		"created_at" INTEGER NOT NULL,
		UNIQUE(post_id, viewer_token)
	);`

	_, err := db.Exec(createSQL)
	return err
}

func cleanupBlogTables() error {
	if _, err := db.Exec(`DROP TABLE IF EXISTS views;`); err != nil {
		return err
	}
	if _, err := db.Exec(`DROP TABLE IF EXISTS blog_post_views;`); err != nil {
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
