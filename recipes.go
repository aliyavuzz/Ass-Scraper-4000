package db

import "database/sql"

// ListRecipes returns all recipes from the database.
func (db *DB) ListRecipes() ([]RecipeRow, error) {
	rows, err := db.Query(`SELECT name, file, site, objective, COALESCE(variables,'[]'), steps, version, COALESCE(cron_expr,''), enabled, created_at, updated_at FROM recipes ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []RecipeRow
	for rows.Next() {
		var r RecipeRow
		if err := rows.Scan(&r.Name, &r.File, &r.Site, &r.Objective, &r.Variables, &r.Steps, &r.Version, &r.CronExpr, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			continue
		}
		recipes = append(recipes, r)
	}
	return recipes, nil
}

// GetRecipe returns a single recipe by name.
func (db *DB) GetRecipe(name string) (*RecipeRow, error) {
	r := &RecipeRow{}
	err := db.QueryRow(`SELECT name, file, site, objective, COALESCE(variables,'[]'), steps, version, COALESCE(cron_expr,''), enabled, created_at, updated_at FROM recipes WHERE name = ?`, name).
		Scan(&r.Name, &r.File, &r.Site, &r.Objective, &r.Variables, &r.Steps, &r.Version, &r.CronExpr, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// SetSchedule updates the cron expression and enabled flag for a recipe.
func (db *DB) SetSchedule(name, cronExpr string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := db.Exec(`UPDATE recipes SET cron_expr=?, enabled=?, updated_at=CURRENT_TIMESTAMP WHERE name=?`, cronExpr, enabledInt, name)
	return err
}
