package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// DB wraps a *sql.DB with convenience methods for Scrape-o-Matic.
type DB struct {
	*sql.DB
}

// InitDB opens (or creates) the SQLite database and runs migrations.
func InitDB(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent reads.
	if _, err := sqlDB.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	return db, nil
}

func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS recipes (
			name        TEXT PRIMARY KEY,
			file        TEXT NOT NULL,
			site        TEXT NOT NULL,
			objective   TEXT NOT NULL,
			variables   TEXT,
			steps       INTEGER,
			version     INTEGER DEFAULT 1,
			cron_expr   TEXT DEFAULT '',
			enabled     INTEGER DEFAULT 1,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS scrape_data (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			recipe_name TEXT NOT NULL,
			data        TEXT NOT NULL,
			success     INTEGER NOT NULL,
			duration    TEXT,
			error       TEXT,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (recipe_name) REFERENCES recipes(name)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scrape_recipe ON scrape_data(recipe_name, created_at DESC)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration error: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

// RecipeRow represents a row in the recipes table.
type RecipeRow struct {
	Name      string `json:"name"`
	File      string `json:"file"`
	Site      string `json:"site"`
	Objective string `json:"objective"`
	Variables string `json:"variables"`
	Steps     int    `json:"steps"`
	Version   int    `json:"version"`
	CronExpr  string `json:"cron_expr"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// DataRow represents a row in the scrape_data table.
type DataRow struct {
	ID         int    `json:"id"`
	RecipeName string `json:"recipe_name"`
	Data       string `json:"data"`
	Success    bool   `json:"success"`
	Duration   string `json:"duration"`
	Error      string `json:"error"`
	CreatedAt  string `json:"created_at"`
}

// SyncRecipesFromDisk scans the recipe directory and upserts into the database.
// Preserves user-set cron_expr and enabled fields.
func (db *DB) SyncRecipesFromDisk(recipeDir string) error {
	entries, err := os.ReadDir(recipeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(recipeDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var flow struct {
			Site      string   `json:"site"`
			Objective string   `json:"objective"`
			Steps     []any    `json:"steps"`
			Variables []string `json:"variables"`
		}
		if err := json.Unmarshal(data, &flow); err != nil {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".json")
		varsJSON, _ := json.Marshal(flow.Variables)

		// Check if exists.
		var existingVersion int
		err = db.QueryRow(`SELECT version FROM recipes WHERE name = ?`, name).Scan(&existingVersion)
		if err == sql.ErrNoRows {
			// Insert new.
			_, err = db.Exec(`INSERT INTO recipes (name, file, site, objective, variables, steps, version) VALUES (?, ?, ?, ?, ?, ?, 1)`,
				name, e.Name(), flow.Site, flow.Objective, string(varsJSON), len(flow.Steps))
			if err != nil {
				return fmt.Errorf("insert recipe %q: %w", name, err)
			}
		} else if err == nil {
			// Update (increment version if content changed).
			_, err = db.Exec(`UPDATE recipes SET file=?, site=?, objective=?, variables=?, steps=?, version=version+1, updated_at=CURRENT_TIMESTAMP WHERE name=?`,
				e.Name(), flow.Site, flow.Objective, string(varsJSON), len(flow.Steps), name)
			if err != nil {
				return fmt.Errorf("update recipe %q: %w", name, err)
			}
		}
	}
	return nil
}
