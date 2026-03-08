package db

import (
	"encoding/json"
)

// SaveScrapeData inserts a scrape result row after every recipe execution.
func (db *DB) SaveScrapeData(recipeName string, data map[string]any, success bool, duration string, errMsg string) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		dataJSON = []byte("{}")
	}
	successInt := 0
	if success {
		successInt = 1
	}
	_, err = db.Exec(`INSERT INTO scrape_data (recipe_name, data, success, duration, error) VALUES (?, ?, ?, ?, ?)`,
		recipeName, string(dataJSON), successInt, duration, errMsg)
	return err
}

// GetLatestData returns the most recent successful scrape result for a recipe.
func (db *DB) GetLatestData(recipeName string) (*DataRow, error) {
	row := &DataRow{}
	err := db.QueryRow(`SELECT id, recipe_name, data, success, COALESCE(duration,''), COALESCE(error,''), created_at FROM scrape_data WHERE recipe_name = ? AND success = 1 ORDER BY created_at DESC LIMIT 1`, recipeName).
		Scan(&row.ID, &row.RecipeName, &row.Data, &row.Success, &row.Duration, &row.Error, &row.CreatedAt)
	if err != nil {
		return nil, err
	}
	return row, nil
}

// GetDataHistory returns the last N scrape results for a recipe.
func (db *DB) GetDataHistory(recipeName string, limit int) ([]DataRow, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.Query(`SELECT id, recipe_name, data, success, COALESCE(duration,''), COALESCE(error,''), created_at FROM scrape_data WHERE recipe_name = ? ORDER BY created_at DESC LIMIT ?`, recipeName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DataRow
	for rows.Next() {
		var r DataRow
		if err := rows.Scan(&r.ID, &r.RecipeName, &r.Data, &r.Success, &r.Duration, &r.Error, &r.CreatedAt); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, nil
}
