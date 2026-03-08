package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/robfig/cron/v3"

	"web-automation-agent/internal/db"
	"web-automation-agent/internal/engine"
)

// Scheduler wraps a cron scheduler and manages background recipe execution.
type Scheduler struct {
	cron    *cron.Cron
	db      *db.DB
	entries map[string]cron.EntryID // recipeName -> entryID
	mu      sync.Mutex
}

// NewScheduler creates a new Scheduler with second-level cron support.
func NewScheduler(database *db.DB) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		db:      database,
		entries: make(map[string]cron.EntryID),
	}
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop gracefully stops the cron scheduler.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// SyncFromDB reads all recipes with non-empty cron_expr and enabled=1,
// then adds/removes cron entries to match.
func (s *Scheduler) SyncFromDB() error {
	recipes, err := s.db.ListRecipes()
	if err != nil {
		return fmt.Errorf("failed to list recipes: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Build a set of recipes that should be scheduled.
	wanted := map[string]string{} // name -> cronExpr
	for _, r := range recipes {
		if r.CronExpr != "" && r.Enabled {
			wanted[r.Name] = r.CronExpr
		}
	}

	// Remove entries that are no longer wanted.
	for name, entryID := range s.entries {
		if _, ok := wanted[name]; !ok {
			s.cron.Remove(entryID)
			delete(s.entries, name)
			engine.LogInfo("SCHED", "Removed schedule for %q", name)
		}
	}

	// Add or update entries.
	for name, expr := range wanted {
		if _, exists := s.entries[name]; exists {
			// Already scheduled — skip (to update, remove first then re-add).
			continue
		}
		recipeName := name
		entryID, err := s.cron.AddFunc(expr, func() {
			s.runRecipe(recipeName)
		})
		if err != nil {
			engine.LogError("SCHED", "Invalid cron %q for %q: %v", expr, name, err)
			continue
		}
		s.entries[name] = entryID
		engine.LogSuccess("SCHED", "Scheduled %q with %q", name, expr)
	}

	return nil
}

// SetSchedule updates a recipe's schedule in the DB and live cron.
func (s *Scheduler) SetSchedule(recipeName, cronExpr string, enabled bool) error {
	// Validate cron expression if non-empty.
	if cronExpr != "" {
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		if _, err := parser.Parse(cronExpr); err != nil {
			return fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
		}
	}

	if err := s.db.SetSchedule(recipeName, cronExpr, enabled); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry.
	if entryID, exists := s.entries[recipeName]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, recipeName)
	}

	// Add new entry if enabled and has cron.
	if cronExpr != "" && enabled {
		entryID, err := s.cron.AddFunc(cronExpr, func() {
			s.runRecipe(recipeName)
		})
		if err != nil {
			return fmt.Errorf("failed to add cron entry: %w", err)
		}
		s.entries[recipeName] = entryID
		engine.LogSuccess("SCHED", "Updated schedule for %q: %s", recipeName, cronExpr)
	} else {
		engine.LogInfo("SCHED", "Disabled schedule for %q", recipeName)
	}

	return nil
}

func (s *Scheduler) runRecipe(recipeName string) {
	engine.LogInfo("SCHED", "Running scheduled recipe: %s", recipeName)

	// Load recipe from disk.
	path := filepath.Join(engine.FlowDir, recipeName+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		engine.LogError("SCHED", "Failed to read recipe %q: %v", recipeName, err)
		s.db.SaveScrapeData(recipeName, nil, false, "", fmt.Sprintf("read error: %v", err))
		return
	}

	var flow engine.FlowFile
	if err := json.Unmarshal(data, &flow); err != nil {
		engine.LogError("SCHED", "Failed to parse recipe %q: %v", recipeName, err)
		s.db.SaveScrapeData(recipeName, nil, false, "", fmt.Sprintf("parse error: %v", err))
		return
	}

	// Run headless with no variables (scheduled runs typically don't have interactive vars).
	result := engine.RunRecipeHeadless(flow, map[string]string{}, false, func(msg string) {
		if !strings.HasPrefix(msg, "__") {
			engine.LogInfo("SCHED", "[%s] %s", recipeName, msg)
		}
	}, nil)

	errMsg := ""
	if !result.Success {
		errMsg = result.Error
	}
	s.db.SaveScrapeData(recipeName, result.Data, result.Success, result.Duration, errMsg)

	if result.Success {
		engine.LogSuccess("SCHED", "Recipe %q completed in %s", recipeName, result.Duration)
	} else {
		engine.LogError("SCHED", "Recipe %q failed: %s", recipeName, result.Error)
	}
}
