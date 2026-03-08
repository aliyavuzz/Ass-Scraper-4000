package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"web-automation-agent/internal/dashboard"
	"web-automation-agent/internal/db"
	"web-automation-agent/internal/engine"
	"web-automation-agent/internal/scheduler"
)

// ============================================================================
// API SERVER
// ============================================================================

// APIServer holds the state for the REST API + dashboard server.
type APIServer struct {
	Port      string
	APIKey    string
	RecipeDir string
	DB        *db.DB
	Scheduler *scheduler.Scheduler
	mu        sync.Mutex
	sessions  map[string]*APISession
}

// APISession tracks a running recipe execution via the API.
type APISession struct {
	ID         string
	Status     string // idle, running, waiting_2fa, done, error
	Logs       []string
	Result     *engine.RecipeResult
	TwoFAChan  chan string
	mu         sync.Mutex
	sseClients map[chan string]bool
	sseMu      sync.Mutex
}

// RecipeInfo is returned by the /api/v1/recipes endpoint.
type RecipeInfo struct {
	Name      string   `json:"name"`
	File      string   `json:"file"`
	Site      string   `json:"site"`
	Objective string   `json:"objective"`
	Variables []string `json:"variables"`
	Steps     int      `json:"steps"`
	CronExpr  string   `json:"cron_expr,omitempty"`
	Enabled   bool     `json:"enabled"`
	Version   int      `json:"version,omitempty"`
}

// RunRequest is the body for POST /api/v1/recipes/{name}/run.
type RunRequest struct {
	Variables  map[string]string `json:"variables"`
	Screenshot bool              `json:"screenshot"`
	Headless   bool              `json:"headless"`
}

func (s *APISession) addLog(msg string) {
	s.mu.Lock()
	s.Logs = append(s.Logs, msg)
	s.mu.Unlock()
	s.broadcast("log", msg)
}

func (s *APISession) broadcast(event, data string) {
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	for ch := range s.sseClients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *APISession) addSSEClient(ch chan string) {
	s.sseMu.Lock()
	s.sseClients[ch] = true
	s.sseMu.Unlock()
}

func (s *APISession) removeSSEClient(ch chan string) {
	s.sseMu.Lock()
	delete(s.sseClients, ch)
	s.sseMu.Unlock()
}

func (srv *APIServer) getSession(id string) *APISession {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.sessions[id]
}

func (srv *APIServer) createSession() *APISession {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	id := fmt.Sprintf("s_%d", time.Now().UnixNano())
	sess := &APISession{
		ID:         id,
		Status:     "idle",
		TwoFAChan:  make(chan string, 1),
		sseClients: make(map[chan string]bool),
	}
	srv.sessions[id] = sess
	return sess
}

// ============================================================================
// API KEY MIDDLEWARE
// ============================================================================

func (srv *APIServer) requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if srv.APIKey == "" {
			next(w, r)
			return
		}
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.URL.Query().Get("api_key")
		}
		if key != srv.APIKey {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// ============================================================================
// HELPERS
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func extractPathParam(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(s, "/"); idx >= 0 {
		return s[:idx]
	}
	return s
}

// ============================================================================
// SERVER STARTUP
// ============================================================================

func CmdServe(args []string) {
	port := "8080"
	apiKey := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-port":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "-key":
			if i+1 < len(args) {
				apiKey = args[i+1]
				i++
			}
		}
	}

	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
	}
	if apiKey == "" {
		engine.LoadEnvFile(".env")
		apiKey = os.Getenv("API_KEY")
	}

	// Initialize database.
	database, err := db.InitDB("scrapematic.db")
	if err != nil {
		log.Fatalf("Database init failed: %v", err)
	}

	// Sync recipes from disk into DB.
	if err := database.SyncRecipesFromDisk(engine.FlowDir); err != nil {
		engine.LogWarning("DB", "Recipe sync failed: %v", err)
	}

	// Initialize and start scheduler.
	sched := scheduler.NewScheduler(database)
	sched.Start()
	defer sched.Stop()

	if err := sched.SyncFromDB(); err != nil {
		engine.LogWarning("SCHED", "Schedule sync failed: %v", err)
	}

	srv := &APIServer{
		Port:      port,
		APIKey:    apiKey,
		RecipeDir: engine.FlowDir,
		DB:        database,
		Scheduler: sched,
		sessions:  make(map[string]*APISession),
	}

	mux := http.NewServeMux()

	// Dashboard (no auth).
	mux.HandleFunc("/", srv.handleIndex)

	// Static files (no auth).
	mux.Handle("/screenshots/", http.StripPrefix("/screenshots/", http.FileServer(http.Dir(engine.ScreenshotDir))))

	// API routes (require auth).
	mux.HandleFunc("/api/v1/recipes", srv.requireAPIKey(srv.handleListRecipes))
	mux.HandleFunc("/api/v1/recipes/", srv.requireAPIKey(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/run") && r.Method == http.MethodPost {
			srv.handleRunRecipe(w, r)
		} else if strings.HasSuffix(path, "/data") {
			srv.handleRecipeData(w, r)
		} else if strings.HasSuffix(path, "/schedule") {
			srv.handleRecipeSchedule(w, r)
		} else {
			srv.handleGetRecipe(w, r)
		}
	}))
	mux.HandleFunc("/api/v1/run", srv.requireAPIKey(srv.handleGenericRun))
	mux.HandleFunc("/api/v1/sessions/", srv.requireAPIKey(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/events") {
			srv.handleSessionEvents(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/2fa") {
			srv.handleSession2FA(w, r)
		} else {
			srv.handleSessionStatus(w, r)
		}
	}))

	// CORS preflight handler.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
			w.WriteHeader(http.StatusNoContent)
		}
	})

	keyStatus := "OFF (no key set)"
	if apiKey != "" {
		keyStatus = "ON (" + apiKey[:min(4, len(apiKey))] + "...)"
	}

	fmt.Printf("\n%s══════════════════════════════════════════════════════════%s\n", engine.ColorGreen, engine.ColorReset)
	fmt.Printf("  %s░▒▓ SCRAPE-O-MATIC 3000 ▓▒░%s\n", engine.ColorBold, engine.ColorReset)
	fmt.Printf("  API Server + Dashboard + Scheduler\n")
	fmt.Printf("  Open: %shttp://localhost:%s%s\n", engine.ColorBold, port, engine.ColorReset)
	fmt.Printf("  API Key Auth: %s\n", keyStatus)
	fmt.Printf("  Recipes dir: %s\n", engine.FlowDir)
	fmt.Printf("  Database: scrapematic.db\n")
	fmt.Printf("%s══════════════════════════════════════════════════════════%s\n\n", engine.ColorGreen, engine.ColorReset)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (srv *APIServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, dashboard.DashboardHTML)
}
