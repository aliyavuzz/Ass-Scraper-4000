package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"web-automation-agent/internal/engine"
)

// ============================================================================
// HTTP HANDLERS
// ============================================================================

func (srv *APIServer) handleListRecipes(w http.ResponseWriter, r *http.Request) {
	// Prefer DB data if available.
	if srv.DB != nil {
		dbRecipes, err := srv.DB.ListRecipes()
		if err == nil && len(dbRecipes) > 0 {
			var infos []RecipeInfo
			for _, r := range dbRecipes {
				var vars []string
				json.Unmarshal([]byte(r.Variables), &vars)
				infos = append(infos, RecipeInfo{
					Name:      r.Name,
					File:      r.File,
					Site:      r.Site,
					Objective: r.Objective,
					Variables: vars,
					Steps:     r.Steps,
					CronExpr:  r.CronExpr,
					Enabled:   r.Enabled,
					Version:   r.Version,
				})
			}
			writeJSON(w, http.StatusOK, infos)
			return
		}
	}

	// Fallback to disk scan.
	recipes, err := srv.listRecipesFromDisk()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, recipes)
}

func (srv *APIServer) listRecipesFromDisk() ([]RecipeInfo, error) {
	entries, err := os.ReadDir(srv.RecipeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RecipeInfo{}, nil
		}
		return nil, err
	}

	var recipes []RecipeInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(srv.RecipeDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var flow engine.FlowFile
		if err := json.Unmarshal(data, &flow); err != nil {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".json")
		recipes = append(recipes, RecipeInfo{
			Name:      name,
			File:      e.Name(),
			Site:      flow.Site,
			Objective: flow.Objective,
			Variables: flow.Variables,
			Steps:     len(flow.Steps),
			Enabled:   true,
		})
	}
	return recipes, nil
}

func (srv *APIServer) loadRecipe(name string) (*engine.FlowFile, error) {
	path := filepath.Join(srv.RecipeDir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("recipe %q not found", name)
	}
	var flow engine.FlowFile
	if err := json.Unmarshal(data, &flow); err != nil {
		return nil, fmt.Errorf("invalid recipe JSON: %w", err)
	}
	return &flow, nil
}

func (srv *APIServer) handleGetRecipe(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r.URL.Path, "/api/v1/recipes/")
	if name == "" || strings.Contains(name, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing recipe name"})
		return
	}
	name = strings.TrimSuffix(name, "/run")
	name = strings.TrimSuffix(name, "/data")
	name = strings.TrimSuffix(name, "/schedule")

	flow, err := srv.loadRecipe(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, flow)
}

func (srv *APIServer) handleRunRecipe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	path := r.URL.Path
	path = strings.TrimPrefix(path, "/api/v1/recipes/")
	path = strings.TrimSuffix(path, "/run")
	recipeName := path

	if recipeName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing recipe name"})
		return
	}

	flow, err := srv.loadRecipe(recipeName)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	var req RunRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}
	if req.Variables == nil {
		req.Variables = map[string]string{}
	}

	if r.Header.Get("Accept") == "text/event-stream" {
		sess := srv.createSession()
		sess.Status = "running"

		go func() {
			result := engine.RunRecipeHeadless(*flow, req.Variables, req.Screenshot, func(msg string) {
				sess.addLog(msg)
				if msg == "__2FA_NEEDED__" {
					sess.mu.Lock()
					sess.Status = "waiting_2fa"
					sess.mu.Unlock()
					sess.broadcast("status", "waiting_2fa")
				}
			}, sess.TwoFAChan)

			sess.mu.Lock()
			sess.Result = &result
			if result.Success {
				sess.Status = "done"
			} else {
				sess.Status = "error"
			}
			sess.mu.Unlock()
			sess.broadcast("status", sess.Status)

			resultJSON, _ := json.Marshal(result)
			sess.broadcast("result", string(resultJSON))

			// Save to DB.
			if srv.DB != nil {
				errMsg := ""
				if !result.Success {
					errMsg = result.Error
				}
				srv.DB.SaveScrapeData(recipeName, result.Data, result.Success, result.Duration, errMsg)
			}
		}()

		writeJSON(w, http.StatusAccepted, map[string]string{
			"session_id": sess.ID,
			"events_url": fmt.Sprintf("/api/v1/sessions/%s/events", sess.ID),
		})
		return
	}

	// Synchronous execution.
	result := engine.RunRecipeHeadless(*flow, req.Variables, req.Screenshot, nil, nil)

	// Save to DB.
	if srv.DB != nil {
		errMsg := ""
		if !result.Success {
			errMsg = result.Error
		}
		srv.DB.SaveScrapeData(recipeName, result.Data, result.Success, result.Duration, errMsg)
	}

	if result.Success {
		writeJSON(w, http.StatusOK, result)
	} else {
		writeJSON(w, http.StatusInternalServerError, result)
	}
}

func (srv *APIServer) handleGenericRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var body struct {
		Recipe     string            `json:"recipe"`
		Variables  map[string]string `json:"variables"`
		Screenshot bool              `json:"screenshot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.Recipe == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "recipe field required"})
		return
	}
	if body.Variables == nil {
		body.Variables = map[string]string{}
	}

	flow, err := srv.loadRecipe(body.Recipe)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	result := engine.RunRecipeHeadless(*flow, body.Variables, body.Screenshot, nil, nil)

	if srv.DB != nil {
		errMsg := ""
		if !result.Success {
			errMsg = result.Error
		}
		srv.DB.SaveScrapeData(body.Recipe, result.Data, result.Success, result.Duration, errMsg)
	}

	if result.Success {
		writeJSON(w, http.StatusOK, result)
	} else {
		writeJSON(w, http.StatusInternalServerError, result)
	}
}

func (srv *APIServer) handleSessionEvents(w http.ResponseWriter, r *http.Request) {
	sessID := extractPathParam(r.URL.Path, "/api/v1/sessions/")
	sessID = strings.TrimSuffix(sessID, "/events")

	sess := srv.getSession(sessID)
	if sess == nil {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := make(chan string, 64)
	sess.addSSEClient(ch)
	defer sess.removeSSEClient(ch)

	sess.mu.Lock()
	for _, l := range sess.Logs {
		fmt.Fprintf(w, "event: log\ndata: %s\n\n", l)
	}
	currentStatus := sess.Status
	sess.mu.Unlock()
	fmt.Fprintf(w, "event: status\ndata: %s\n\n", currentStatus)
	flusher.Flush()

	if currentStatus == "done" || currentStatus == "error" {
		sess.mu.Lock()
		if sess.Result != nil {
			resultJSON, _ := json.Marshal(sess.Result)
			fmt.Fprintf(w, "event: result\ndata: %s\n\n", string(resultJSON))
		}
		sess.mu.Unlock()
		flusher.Flush()
		return
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			fmt.Fprint(w, msg)
			flusher.Flush()
		}
	}
}

func (srv *APIServer) handleSession2FA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	sessID := extractPathParam(r.URL.Path, "/api/v1/sessions/")
	sessID = strings.TrimSuffix(sessID, "/2fa")

	sess := srv.getSession(sessID)
	if sess == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code field required"})
		return
	}

	select {
	case sess.TwoFAChan <- body.Code:
		sess.mu.Lock()
		sess.Status = "running"
		sess.mu.Unlock()
		sess.broadcast("status", "running")
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		writeJSON(w, http.StatusConflict, map[string]string{"error": "not waiting for 2FA"})
	}
}

func (srv *APIServer) handleSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessID := extractPathParam(r.URL.Path, "/api/v1/sessions/")
	sessID = strings.TrimSuffix(sessID, "/status")

	sess := srv.getSession(sessID)
	if sess == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	resp := map[string]any{
		"id":     sess.ID,
		"status": sess.Status,
		"logs":   sess.Logs,
	}
	if sess.Result != nil {
		resp["result"] = sess.Result
	}
	writeJSON(w, http.StatusOK, resp)
}

// ============================================================================
// DATA & SCHEDULE ENDPOINTS
// ============================================================================

func (srv *APIServer) handleRecipeData(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r.URL.Path, "/api/v1/recipes/")
	name = strings.TrimSuffix(name, "/data")

	if srv.DB == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
		return
	}

	// Check for history param.
	historyStr := r.URL.Query().Get("history")
	if historyStr != "" {
		limit, _ := strconv.Atoi(historyStr)
		if limit <= 0 {
			limit = 20
		}
		rows, err := srv.DB.GetDataHistory(name, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, rows)
		return
	}

	row, err := srv.DB.GetLatestData(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no data found for recipe"})
		return
	}

	// Parse the JSON data string back into an object.
	var data any
	json.Unmarshal([]byte(row.Data), &data)

	writeJSON(w, http.StatusOK, map[string]any{
		"recipe":     name,
		"data":       data,
		"scraped_at": row.CreatedAt,
		"duration":   row.Duration,
	})
}

func (srv *APIServer) handleRecipeSchedule(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r.URL.Path, "/api/v1/recipes/")
	name = strings.TrimSuffix(name, "/schedule")

	if srv.DB == nil || srv.Scheduler == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "scheduler not available"})
		return
	}

	if r.Method == http.MethodGet {
		recipe, err := srv.DB.GetRecipe(name)
		if err != nil || recipe == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "recipe not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"recipe":  name,
			"cron":    recipe.CronExpr,
			"enabled": recipe.Enabled,
		})
		return
	}

	if r.Method == http.MethodPut || r.Method == http.MethodPost {
		var body struct {
			Cron    string `json:"cron"`
			Enabled *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		enabled := true
		if body.Enabled != nil {
			enabled = *body.Enabled
		}

		if err := srv.Scheduler.SetSchedule(name, body.Cron, enabled); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"recipe":  name,
			"cron":    body.Cron,
			"enabled": enabled,
			"status":  "updated",
		})
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or PUT required"})
}
