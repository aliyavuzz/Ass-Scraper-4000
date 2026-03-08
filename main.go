package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"web-automation-agent/internal/engine"
	"web-automation-agent/internal/server"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			cmdList()
			return
		case "run":
			cmdRun(os.Args[2:])
			return
		case "manual":
			cmdManual(os.Args[2:])
			return
		case "learn":
			cmdLearn(os.Args[2:])
			return
		case "screenshot":
			cmdScreenshot(os.Args[2:])
			return
		case "interactive":
			cmdInteractive()
			return
		case "serve":
			server.CmdServe(os.Args[2:])
			return
		}
	}
	printUsage()
}

func printUsage() {
	fmt.Println(`
Scrape-o-Matic 3000 — No-Code Scrape-to-API Platform

USAGE:
  agent list                              List entries from credentials.csv
  agent run [flags]                       Run automation (AI record + replay)
  agent manual [flags]                    Manual mode — type commands to control browser
  agent learn [flags]                     Learning mode — click in Chrome, agent records
  agent serve [flags]                     Start the API server + web dashboard
  agent screenshot [flags]                Take a screenshot of a URL
  agent interactive                       Interactive mode (prompts for input)

RUN FLAGS:
  -carrier    string   Entry name from CSV
  -tenant     string   Tenant filter — optional
  -url        string   Direct URL (overrides CSV lookup)
  -user       string   Username
  -pass       string   Password
  -objective  string   What to do / recipe label (default: "login")
  -csv        string   Path to credentials CSV (default: "credentials.csv")
  -key        string   Gemini API key (run only; or set GEMINI_API_KEY env var)
  -headless   bool     Run browser in headless mode (default: false)
  -screenshot bool     Take screenshot after completion (default: true)

SERVE FLAGS:
  -port       string   Port for API server (default: "8080")
  -key        string   API key for authentication (or set API_KEY env var)

MANUAL/LEARN FLAGS: (same as run, minus -key — no AI needed)
  -carrier, -tenant, -url, -user, -pass, -objective, -csv, -screenshot

ACTIONS SUPPORTED:
  CLICK, TYPE, SELECT, HOVER, UPLOAD, SCRAPE, SCROLL,
  PRESS_KEY, WAIT_URL, WAIT_ELEMENT, WAIT_IDLE, WAIT_2FA,
  SCREENSHOT, DONE

EXAMPLES:
  agent learn -url "https://example.com/login" -objective "login"
  agent manual -url "https://example.com" -user admin -pass secret
  agent run -url "https://example.com" -objective "login" -headless
  agent serve -port 8080 -key my-secret-key
  agent screenshot -url "https://example.com"`)
	fmt.Println()
}

func cmdList() {
	csvPath := "credentials.csv"
	if len(os.Args) > 2 {
		csvPath = os.Args[2]
	}
	creds, err := engine.LoadCredentials(csvPath)
	if err != nil {
		log.Fatalf("Failed to load credentials: %v", err)
	}
	engine.ListCredentials(creds)
}

func cmdRun(args []string) {
	flags, _ := engine.ParseCommonFlags("run", args, true)

	key := engine.GetAPIKey(flags.APIKey)
	if key == "" {
		log.Fatal("No Gemini API key provided. Use -key flag, GEMINI_API_KEY env var, or .env file.")
	}
	flags.APIKey = key

	config, err := engine.ResolveConfig(flags)
	if err != nil {
		log.Fatal(err)
	}
	config.GeminiKey = key

	if err := engine.RunAgent(config); err != nil {
		log.Fatalf("%s[FATAL]%s Agent failed: %v\n", engine.ColorRed, engine.ColorReset, err)
	}
}

func cmdManual(args []string) {
	flags, _ := engine.ParseCommonFlags("manual", args, false)
	config, err := engine.ResolveConfig(flags)
	if err != nil {
		log.Fatal(err)
	}

	browser, page, err := engine.InitBrowser(config.Headless)
	if err != nil {
		log.Fatalf("Browser init failed: %v", err)
	}
	defer browser.Close()

	if err := engine.NavigateTo(page, config.StartURL); err != nil {
		log.Fatalf("Navigation failed: %v", err)
	}

	if err := engine.RunManualMode(page, config); err != nil {
		log.Fatalf("%s[FATAL]%s Manual mode failed: %v\n", engine.ColorRed, engine.ColorReset, err)
	}

	if config.Screenshot {
		parsed, _ := url.Parse(config.StartURL)
		engine.TakeScreenshot(page, parsed.Hostname()+"_"+config.Objective)
	}
}

func cmdLearn(args []string) {
	flags, _ := engine.ParseCommonFlags("learn", args, false)
	flags.Headless = false // Learning mode must be visible.
	config, err := engine.ResolveConfig(flags)
	if err != nil {
		log.Fatal(err)
	}
	config.Headless = false

	browser, page, err := engine.InitBrowser(false)
	if err != nil {
		log.Fatalf("Browser init failed: %v", err)
	}
	defer browser.Close()

	if err := engine.RunLearnMode(page, config); err != nil {
		log.Fatalf("%s[FATAL]%s Learning mode failed: %v\n", engine.ColorRed, engine.ColorReset, err)
	}

	if config.Screenshot {
		parsed, _ := url.Parse(config.StartURL)
		engine.TakeScreenshot(page, parsed.Hostname()+"_"+config.Objective)
	}
}

func cmdScreenshot(args []string) {
	fs := flag.NewFlagSet("screenshot", flag.ExitOnError)
	targetURL := fs.String("url", "", "URL to screenshot")
	out := fs.String("out", "", "Output filename label")
	headless := fs.Bool("headless", false, "Headless browser mode")
	fs.Parse(args)

	if *targetURL == "" {
		log.Fatal("Provide -url for screenshot command.")
	}
	if !strings.HasPrefix(*targetURL, "http") {
		*targetURL = "https://" + *targetURL
	}

	browser, page, err := engine.InitBrowser(*headless)
	if err != nil {
		log.Fatalf("Browser init failed: %v", err)
	}
	defer browser.Close()

	if err := engine.NavigateTo(page, *targetURL); err != nil {
		log.Fatalf("Navigation failed: %v", err)
	}

	label := *out
	if label == "" {
		parsed, _ := url.Parse(*targetURL)
		label = parsed.Hostname()
	}

	path, err := engine.TakeScreenshot(page, label)
	if err != nil {
		log.Fatalf("Screenshot failed: %v", err)
	}
	engine.LogSuccess("DONE", "Screenshot saved to %s", path)
}

func cmdInteractive() {
	reader := bufio.NewReader(os.Stdin)

	key := engine.GetAPIKey("")
	if key == "" {
		fmt.Print("Gemini API Key: ")
		key, _ = reader.ReadString('\n')
		key = strings.TrimSpace(key)
	}
	if key == "" {
		log.Fatal("API key is required.")
	}

	fmt.Print("URL to automate: ")
	startURL, _ := reader.ReadString('\n')
	startURL = strings.TrimSpace(startURL)
	if startURL == "" {
		log.Fatal("URL is required.")
	}
	if !strings.HasPrefix(startURL, "http") {
		startURL = "https://" + startURL
	}

	fmt.Print("Objective (what should the agent do?): ")
	objective, _ := reader.ReadString('\n')
	objective = strings.TrimSpace(objective)
	if objective == "" {
		objective = "login"
	}

	variables := map[string]string{}
	fmt.Print("Username (leave empty to skip): ")
	u, _ := reader.ReadString('\n')
	u = strings.TrimSpace(u)
	if u != "" {
		variables["username"] = u
	}

	fmt.Print("Password (leave empty to skip): ")
	p, _ := reader.ReadString('\n')
	p = strings.TrimSpace(p)
	if p != "" {
		variables["password"] = p
	}

	fmt.Print("Any extra variables? (key=value, comma separated, or empty): ")
	extras, _ := reader.ReadString('\n')
	extras = strings.TrimSpace(extras)
	if extras != "" {
		for _, pair := range strings.Split(extras, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(parts) == 2 {
				variables[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	fmt.Print("Take screenshot after completion? (y/n, default y): ")
	ssInput, _ := reader.ReadString('\n')
	ssInput = strings.TrimSpace(strings.ToLower(ssInput))
	screenshot := ssInput != "n"

	config := engine.AgentConfig{
		StartURL: startURL, Objective: objective, Variables: variables,
		GeminiKey: key, Headless: false, Screenshot: screenshot,
	}

	fmt.Println()
	engine.LogInfo("START", "URL: %s", startURL)
	engine.LogInfo("START", "Objective: %s", objective)
	engine.LogInfo("START", "Variables: %d defined", len(variables))
	fmt.Println()

	if err := engine.RunAgent(config); err != nil {
		log.Fatalf("%s[FATAL]%s Agent failed: %v\n", engine.ColorRed, engine.ColorReset, err)
	}
}
