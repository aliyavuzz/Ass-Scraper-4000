package engine

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// ============================================================================
// FLOW MANAGER
// ============================================================================

func FlowFileName(site, objective string) string {
	return filepath.Join(FlowDir, fmt.Sprintf("%s_%s.json", SanitizeFilename(site), SanitizeFilename(objective)))
}

func SanitizeFilename(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	return re.ReplaceAllString(s, "_")
}

func LoadFlow(site, objective string) (*FlowFile, error) {
	path := FlowFileName(site, objective)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read flow file: %w", err)
	}
	var flow FlowFile
	if err := json.Unmarshal(data, &flow); err != nil {
		return nil, fmt.Errorf("failed to parse flow file: %w", err)
	}
	LogInfo("FLOW", "Loaded flow from %s (%d steps)", path, len(flow.Steps))
	return &flow, nil
}

func SaveFlow(flow FlowFile) error {
	if err := os.MkdirAll(FlowDir, 0o755); err != nil {
		return fmt.Errorf("failed to create flows directory: %w", err)
	}
	path := FlowFileName(flow.Site, flow.Objective)
	data, err := json.MarshalIndent(flow, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal flow: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write flow file: %w", err)
	}
	LogSuccess("FLOW", "Saved to %s", path)
	return nil
}

func TemplateSteps(steps []ActionStep, variables map[string]string) []ActionStep {
	templated := make([]ActionStep, len(steps))
	for i, s := range steps {
		templated[i] = ActionStep{
			Action: s.Action, Index: s.Index, X: s.X, Y: s.Y,
			Value: ReverseSubstituteVariables(s.Value, variables),
		}
	}
	return templated
}

func ExtractVariableNames(steps []ActionStep) []string {
	seen := make(map[string]bool)
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	for _, s := range steps {
		for _, m := range re.FindAllStringSubmatch(s.Value, -1) {
			seen[m[1]] = true
		}
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	return names
}

// ============================================================================
// ERROR & RECOVERY
// ============================================================================

func ClassifyError(err error, page *rod.Page, expectedElementCount, consecutiveFailures int) ErrorLevel {
	if err == nil {
		return RetryAction
	}
	msg := err.Error()
	if strings.Contains(msg, "websocket") || strings.Contains(msg, "disconnected") {
		return Fatal
	}
	if strings.Contains(msg, "401") || strings.Contains(msg, "403") {
		return Fatal
	}
	if consecutiveFailures >= MaxRetries {
		return RegenerateFlow
	}
	if expectedElementCount > 0 && page != nil {
		_, _, cur, e := GetUIMap(page)
		if e == nil && cur > 0 {
			diff := float64(intAbs(cur-expectedElementCount)) / float64(expectedElementCount)
			if diff > StaleThreshold {
				LogWarning("RECOVERY", "Layout change detected (was %d elements, now %d)", expectedElementCount, cur)
				return RegenerateFlow
			}
		}
	}
	if strings.Contains(msg, "ElementFromPoint") {
		return RegenerateFlow
	}
	if strings.Contains(msg, "WAIT_URL timeout") || strings.Contains(msg, "WAIT_ELEMENT timeout") {
		return RegenerateFlow
	}
	return RetryAction
}

func intAbs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func RecoveryAction(page *rod.Page, config AgentConfig) error {
	LogWarning("RECOVERY", "Flow outdated or element not found. Switching to Record Mode.")
	parsed, _ := url.Parse(config.StartURL)
	_ = os.Remove(FlowFileName(parsed.Hostname(), config.Objective))
	if err := NavigateTo(page, config.StartURL); err != nil {
		return fmt.Errorf("recovery navigation failed: %w", err)
	}
	return RunRecordMode(page, config)
}

// ============================================================================
// AGENTIC LOOP (AI Record + Replay)
// ============================================================================

func RunAgent(config AgentConfig) error {
	browser, page, err := InitBrowser(config.Headless)
	if err != nil {
		return fmt.Errorf("browser init failed: %w", err)
	}
	defer browser.Close()

	parsed, err := url.Parse(config.StartURL)
	if err != nil {
		return fmt.Errorf("invalid start URL: %w", err)
	}
	hostname := parsed.Hostname()

	var navErr error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		navErr = NavigateTo(page, config.StartURL)
		if navErr == nil {
			break
		}
		LogError("BROWSER", "Navigation attempt %d/%d failed: %v", attempt, MaxRetries, navErr)
		time.Sleep(2 * time.Second)
	}
	if navErr != nil {
		return fmt.Errorf("unable to navigate to start URL after %d attempts: %w", MaxRetries, navErr)
	}

	flow, err := LoadFlow(hostname, config.Objective)
	if err != nil {
		LogWarning("FLOW", "Failed to load flow file: %v — entering Record Mode", err)
		flow = nil
	}

	if flow != nil {
		_, _, currentCount, mapErr := GetUIMap(page)
		if mapErr == nil && len(flow.PageSignatures) > 0 && flow.PageSignatures[0] > 0 {
			diff := float64(intAbs(currentCount-flow.PageSignatures[0])) / float64(flow.PageSignatures[0])
			if diff > StaleThreshold {
				LogWarning("RECOVERY", "Layout change detected on initial page (was %d, now %d). Skipping replay.", flow.PageSignatures[0], currentCount)
				flow = nil
			}
		}
	}

	if flow != nil {
		LogInfo("REPLAY", "Entering Replay Mode (%d steps)", len(flow.Steps))
		if err := RunReplayMode(page, config, flow); err != nil {
			LogWarning("REPLAY", "Replay failed: %v", err)
			return RecoveryAction(page, config)
		}
		if config.Screenshot {
			TakeScreenshot(page, hostname+"_"+config.Objective)
		}
		return nil
	}

	LogInfo("RECORD", "Entering Record Mode for objective: %s", config.Objective)
	if err := RunRecordMode(page, config); err != nil {
		return err
	}
	if config.Screenshot {
		TakeScreenshot(page, hostname+"_"+config.Objective)
	}
	return nil
}

func RunReplayMode(page *rod.Page, config AgentConfig, flow *FlowFile) error {
	totalSteps := len(flow.Steps)
	pageIndex := 0
	for i, step := range flow.Steps {
		LogInfo("REPLAY", "Executing step %d/%d: %s", i+1, totalSteps, FormatStep(step))
		retries := 0
		consecutiveFailures := 0
		for {
			err := ExecuteStep(page, step, nil, config.Variables, &config)
			if err == nil {
				break
			}
			retries++
			consecutiveFailures++
			LogError("ERROR", "Step %d failed (attempt %d/%d): %v", i+1, retries, MaxRetries, err)
			expectedCount := 0
			if pageIndex < len(flow.PageSignatures) {
				expectedCount = flow.PageSignatures[pageIndex]
			}
			level := ClassifyError(err, page, expectedCount, consecutiveFailures)
			switch level {
			case RetryAction:
				if retries >= MaxRetries {
					return fmt.Errorf("step %d exhausted retries: %w", i+1, err)
				}
				time.Sleep(1 * time.Second)
			case RegenerateFlow:
				return fmt.Errorf("step %d requires flow regeneration: %w", i+1, err)
			case Fatal:
				return fmt.Errorf("fatal error at step %d: %w", i+1, err)
			}
		}
		if strings.ToUpper(step.Action) == "DONE" {
			break
		}
		if strings.ToUpper(step.Action) == "CLICK" || strings.ToUpper(step.Action) == "WAIT_URL" {
			pageIndex++
		}
	}
	LogSuccess("SUCCESS", "Objective '%s' completed in %d steps (Replay Mode)", config.Objective, totalSteps)
	return nil
}

func RunRecordMode(page *rod.Page, config AgentConfig) error {
	parsed, _ := url.Parse(config.StartURL)
	hostname := parsed.Hostname()
	var allSteps []ActionStep
	var pageSignatures []int

	for iter := 0; iter < MaxLoopIter; iter++ {
		WaitForPageReady(page)
		elements, uiMap, count, err := GetUIMap(page)
		if err != nil {
			return fmt.Errorf("UI map extraction failed: %w", err)
		}
		pageSignatures = append(pageSignatures, count)
		pageInfo, _ := page.Info()
		currentURL := ""
		if pageInfo != nil {
			currentURL = pageInfo.URL
		}

		llmResp, err := AskLLM(config.GeminiKey, uiMap, config.Objective, config.Variables, currentURL)
		if err != nil {
			return fmt.Errorf("LLM call failed: %w", err)
		}

		done := false
		for _, step := range llmResp.Steps {
			if strings.ToUpper(step.Action) == "DONE" {
				allSteps = append(allSteps, step)
				done = true
				break
			}
			if strings.ToUpper(step.Action) == "SCREENSHOT" {
				label := step.Value
				if label == "" {
					label = hostname + "_step"
				}
				TakeScreenshot(page, label)
				allSteps = append(allSteps, step)
				continue
			}
			if step.Index > 0 && step.Index <= len(elements) {
				step.X = elements[step.Index-1].X
				step.Y = elements[step.Index-1].Y
			}
			retries := 0
			var execErr error
			for retries < MaxRetries {
				execErr = ExecuteStep(page, step, elements, config.Variables)
				if execErr == nil {
					break
				}
				retries++
				LogError("ERROR", "Step execution failed (attempt %d/%d): %v", retries, MaxRetries, execErr)
				time.Sleep(1 * time.Second)
			}
			if execErr != nil {
				LogWarning("RECOVERY", "Step failed after %d retries, re-asking LLM with error context", MaxRetries)
				break
			}
			allSteps = append(allSteps, ActionStep{
				Action: step.Action, Index: step.Index, X: step.X, Y: step.Y, Value: step.Value,
			})
		}
		if done || llmResp.GoalReached {
			break
		}
	}

	templatedSteps := TemplateSteps(allSteps, config.Variables)
	flow := FlowFile{
		Site: hostname, Objective: config.Objective,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		PageSignatures: pageSignatures, Steps: templatedSteps,
		Variables: ExtractVariableNames(templatedSteps),
	}
	if err := SaveFlow(flow); err != nil {
		return err
	}
	LogSuccess("SUCCESS", "Objective '%s' completed in %d steps (Record Mode)", config.Objective, len(allSteps))
	return nil
}

// ============================================================================
// RECIPE EXECUTION (headless, used by API server)
// ============================================================================

// RunRecipeHeadless executes a flow headlessly and returns structured results.
func RunRecipeHeadless(flow FlowFile, variables map[string]string, wantScreenshot bool, statusFn func(string), twofaChan chan string) RecipeResult {
	start := time.Now()
	emit := func(msg string) {
		if statusFn != nil {
			statusFn(msg)
		}
	}

	emit("Launching headless browser (stealth mode)...")
	browser, page, err := InitBrowser(true)
	if err != nil {
		return RecipeResult{Error: fmt.Sprintf("browser init failed: %v", err), Duration: time.Since(start).String()}
	}
	defer browser.Close()

	startURL := "https://" + flow.Site
	emit(fmt.Sprintf("Navigating to %s", startURL))
	if err := NavigateTo(page, startURL); err != nil {
		return RecipeResult{Error: fmt.Sprintf("navigation failed: %v", err), Duration: time.Since(start).String()}
	}

	data := map[string]any{}
	stepsExecuted := 0

	config := AgentConfig{
		StartURL:  startURL,
		Objective: flow.Objective,
		Variables: variables,
		Headless:  true,
		OnStatus:  statusFn,
		TwoFAChan: twofaChan,
	}

	for i, step := range flow.Steps {
		if strings.ToUpper(step.Action) == "DONE" {
			stepsExecuted = i + 1
			emit("DONE — objective achieved")
			break
		}

		emit(fmt.Sprintf("[%d/%d] %s", i+1, len(flow.Steps), FormatStep(step)))
		err := ExecuteStep(page, step, nil, variables, &config)
		if err != nil {
			return RecipeResult{
				Steps:    i,
				Data:     data,
				Error:    fmt.Sprintf("step %d failed: %v", i+1, err),
				Duration: time.Since(start).String(),
			}
		}

		// Collect SCRAPE results.
		if strings.ToUpper(step.Action) == "SCRAPE" && config.LastScrapeResult != nil {
			switch v := config.LastScrapeResult.(type) {
			case map[string]any:
				for k, val := range v {
					data[k] = val
				}
			default:
				data[fmt.Sprintf("scrape_%d", i)] = v
			}
			config.LastScrapeResult = nil
		}

		stepsExecuted = i + 1
		time.Sleep(300 * time.Millisecond)
	}

	result := RecipeResult{
		Success:  true,
		Steps:    stepsExecuted,
		Data:     data,
		Duration: time.Since(start).String(),
	}

	if wantScreenshot {
		emit("Taking screenshot...")
		screenshotBytes, err := page.Screenshot(true, nil)
		if err == nil {
			encoded := "data:image/png;base64," + base64.StdEncoding.EncodeToString(screenshotBytes)
			result.Screenshot = encoded
		}
	}

	return result
}
