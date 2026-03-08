package engine

import (
	"flag"
	"fmt"
	"strings"
)

// ============================================================================
// SHARED CLI HELPERS
// ============================================================================

func ParseCommonFlags(name string, args []string, includeAPIKey bool) (CommonFlags, *flag.FlagSet) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	flags := CommonFlags{}

	carrier := fs.String("carrier", "", "Carrier name from CSV")
	tenant := fs.String("tenant", "", "Tenant filter")
	directURL := fs.String("url", "", "Direct URL (overrides CSV)")
	user := fs.String("user", "", "Username override")
	pass := fs.String("pass", "", "Password override")
	objective := fs.String("objective", "login", "Objective / flow label")
	csvPath := fs.String("csv", "credentials.csv", "Path to credentials CSV")
	headless := fs.Bool("headless", false, "Headless browser mode")
	screenshot := fs.Bool("screenshot", true, "Take screenshot after completion")

	var apiKey *string
	if includeAPIKey {
		apiKey = fs.String("key", "", "Gemini API key")
	}

	fs.Parse(args)

	flags.Carrier = *carrier
	flags.Tenant = *tenant
	flags.DirectURL = *directURL
	flags.User = *user
	flags.Pass = *pass
	flags.Objective = *objective
	flags.CSVPath = *csvPath
	flags.Headless = *headless
	flags.Screenshot = *screenshot
	if apiKey != nil {
		flags.APIKey = *apiKey
	}
	return flags, fs
}

func ResolveConfig(flags CommonFlags) (AgentConfig, error) {
	var startURL, username, password string

	if flags.DirectURL != "" {
		startURL = flags.DirectURL
		username = flags.User
		password = flags.Pass
	} else if flags.Carrier != "" {
		creds, err := LoadCredentials(flags.CSVPath)
		if err != nil {
			return AgentConfig{}, fmt.Errorf("failed to load credentials: %w", err)
		}
		cred := FindCredential(creds, flags.Carrier, flags.Tenant)
		if cred == nil {
			return AgentConfig{}, fmt.Errorf("carrier %q (tenant %q) not found in %s", flags.Carrier, flags.Tenant, flags.CSVPath)
		}
		if cred.URL == "" {
			return AgentConfig{}, fmt.Errorf("carrier %q has no URL in CSV", flags.Carrier)
		}
		startURL = cred.URL
		username = cred.AdminUserName
		password = cred.AdminPassword
		if flags.User != "" {
			username = flags.User
		}
		if flags.Pass != "" {
			password = flags.Pass
		}
		LogInfo("CSV", "Carrier: %s | Tenant: %s | URL: %s", cred.Carrier, cred.Tenant, startURL)
	} else {
		return AgentConfig{}, fmt.Errorf("provide either -carrier or -url")
	}

	if !strings.HasPrefix(startURL, "http") {
		startURL = "https://" + startURL
	}

	variables := map[string]string{}
	if username != "" {
		variables["username"] = username
	}
	if password != "" {
		variables["password"] = password
	}

	return AgentConfig{
		StartURL:   startURL,
		Objective:  flags.Objective,
		Variables:  variables,
		GeminiKey:  flags.APIKey,
		Headless:   flags.Headless,
		Screenshot: flags.Screenshot,
	}, nil
}
