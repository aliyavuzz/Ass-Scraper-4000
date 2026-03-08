package engine

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// ============================================================================
// CSV LOADER
// ============================================================================

func LoadCredentials(path string) ([]Credential, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("CSV has no data rows")
	}

	var creds []Credential
	for _, row := range rows[1:] {
		if len(row) < 7 {
			continue
		}
		u := strings.TrimSpace(row[6])
		if u != "" && !strings.HasPrefix(u, "http") {
			u = "https://" + u
		}
		creds = append(creds, Credential{
			Carrier:       strings.TrimSpace(row[0]),
			Tenant:        strings.TrimSpace(row[1]),
			AdminUserName: strings.TrimSpace(row[2]),
			AdminPassword: strings.TrimSpace(row[3]),
			AdminCode:     strings.TrimSpace(row[4]),
			AdminUser:     strings.TrimSpace(row[5]),
			URL:           u,
		})
	}
	return creds, nil
}

func ListCredentials(creds []Credential) {
	LogInfo("CSV", "Available carriers (%d):", len(creds))
	for i, c := range creds {
		tenant := c.Tenant
		if tenant == "" {
			tenant = "-"
		}
		urlShort := c.URL
		if urlShort == "" {
			urlShort = "(no URL)"
		}
		fmt.Printf("  %s[%2d]%s %-20s tenant=%-6s %s\n", ColorCyan, i+1, ColorReset, c.Carrier, tenant, urlShort)
	}
}

func FindCredential(creds []Credential, carrier, tenant string) *Credential {
	carrier = strings.ToLower(strings.TrimSpace(carrier))
	tenant = strings.ToLower(strings.TrimSpace(tenant))
	for _, c := range creds {
		if strings.ToLower(strings.TrimSpace(c.Carrier)) == carrier {
			if tenant == "" || strings.ToLower(strings.TrimSpace(c.Tenant)) == tenant {
				return &c
			}
		}
	}
	return nil
}
