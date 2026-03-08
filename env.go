package engine

import (
	"os"
	"strings"
)

// LoadEnvFile reads a .env file and sets environment variables.
func LoadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}

func GetAPIKey(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv("GEMINI_API_KEY"); v != "" {
		return v
	}
	LoadEnvFile(".env")
	if v := os.Getenv("GEMINI_API_KEY"); v != "" {
		return v
	}
	return ""
}
