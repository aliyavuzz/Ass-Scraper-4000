package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// ============================================================================
// MANUAL MODE
// ============================================================================

// ParseManualCommand parses a single terminal command into an ActionStep.
// Returns (step, isRefresh, error). isRefresh=true means re-scan the UI map.
func ParseManualCommand(input string, maxIndex int) (ActionStep, bool, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return ActionStep{}, false, fmt.Errorf("")
	}

	parts := strings.SplitN(input, " ", 3)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "click":
		if len(parts) < 2 {
			return ActionStep{}, false, fmt.Errorf("usage: click <index>")
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil || idx < 1 || idx > maxIndex {
			return ActionStep{}, false, fmt.Errorf("index must be between 1 and %d", maxIndex)
		}
		return ActionStep{Action: "CLICK", Index: idx}, false, nil

	case "type":
		if len(parts) < 3 {
			return ActionStep{}, false, fmt.Errorf("usage: type <index> <value>")
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil || idx < 1 || idx > maxIndex {
			return ActionStep{}, false, fmt.Errorf("index must be between 1 and %d", maxIndex)
		}
		return ActionStep{Action: "TYPE", Index: idx, Value: parts[2]}, false, nil

	case "select":
		if len(parts) < 3 {
			return ActionStep{}, false, fmt.Errorf("usage: select <index> <value>")
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil || idx < 1 || idx > maxIndex {
			return ActionStep{}, false, fmt.Errorf("index must be between 1 and %d", maxIndex)
		}
		return ActionStep{Action: "SELECT", Index: idx, Value: parts[2]}, false, nil

	case "scroll":
		return ActionStep{Action: "SCROLL"}, false, nil

	case "wait_url":
		if len(parts) < 2 {
			return ActionStep{}, false, fmt.Errorf("usage: wait_url <substring>")
		}
		val := strings.TrimSpace(strings.TrimPrefix(input, parts[0]))
		return ActionStep{Action: "WAIT_URL", Value: val}, false, nil

	case "wait_element":
		if len(parts) < 2 {
			return ActionStep{}, false, fmt.Errorf("usage: wait_element <text>")
		}
		val := strings.TrimSpace(strings.TrimPrefix(input, parts[0]))
		return ActionStep{Action: "WAIT_ELEMENT", Value: val}, false, nil

	case "screenshot":
		label := ""
		if len(parts) >= 2 {
			label = parts[1]
		}
		return ActionStep{Action: "SCREENSHOT", Value: label}, false, nil

	case "wait_2fa", "2fa":
		if len(parts) < 2 {
			return ActionStep{}, false, fmt.Errorf("usage: wait_2fa <index>")
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil || idx < 1 || idx > maxIndex {
			return ActionStep{}, false, fmt.Errorf("index must be between 1 and %d", maxIndex)
		}
		return ActionStep{Action: "WAIT_2FA", Index: idx}, false, nil

	case "hover":
		if len(parts) < 2 {
			return ActionStep{}, false, fmt.Errorf("usage: hover <index>")
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil || idx < 1 || idx > maxIndex {
			return ActionStep{}, false, fmt.Errorf("index must be between 1 and %d", maxIndex)
		}
		return ActionStep{Action: "HOVER", Index: idx}, false, nil

	case "upload":
		if len(parts) < 3 {
			return ActionStep{}, false, fmt.Errorf("usage: upload <index> <filepath>")
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil || idx < 1 || idx > maxIndex {
			return ActionStep{}, false, fmt.Errorf("index must be between 1 and %d", maxIndex)
		}
		return ActionStep{Action: "UPLOAD", Index: idx, Value: parts[2]}, false, nil

	case "scrape":
		if len(parts) < 2 {
			return ActionStep{}, false, fmt.Errorf("usage: scrape <css_selector_or_json_schema>")
		}
		val := strings.TrimSpace(strings.TrimPrefix(input, parts[0]))
		return ActionStep{Action: "SCRAPE", Value: val}, false, nil

	case "wait_idle":
		return ActionStep{Action: "WAIT_IDLE"}, false, nil

	case "press", "press_key":
		if len(parts) < 2 {
			return ActionStep{}, false, fmt.Errorf("usage: press <key>")
		}
		return ActionStep{Action: "PRESS_KEY", Value: parts[1]}, false, nil

	case "done":
		return ActionStep{Action: "DONE"}, false, nil

	case "refresh":
		return ActionStep{}, true, nil

	default:
		return ActionStep{}, false, fmt.Errorf("unknown command %q. Available: click, type, select, scroll, hover, upload, scrape, press, wait_url, wait_element, wait_idle, wait_2fa, screenshot, refresh, done", cmd)
	}
}

func PrintManualBanner(currentURL string, elementCount int) {
	fmt.Printf("\n%s══════════════════════════════════════════════════════════%s\n", ColorCyan, ColorReset)
	fmt.Printf("%s  MANUAL MODE%s — %s\n", ColorBold, ColorReset, currentURL)
	fmt.Printf("%s  Elements: %d interactive%s\n", ColorCyan, elementCount, ColorReset)
	fmt.Printf("%s══════════════════════════════════════════════════════════%s\n\n", ColorCyan, ColorReset)
}

func PrintManualHelp() {
	fmt.Printf("%sCommands:%s\n", ColorYellow, ColorReset)
	fmt.Println("  click <index>              Click element by index")
	fmt.Println("  type <index> <value>       Type value into element")
	fmt.Println("  select <index> <value>     Select option in dropdown")
	fmt.Println("  hover <index>              Hover over element")
	fmt.Println("  upload <index> <path>      Upload file to input element")
	fmt.Println("  scrape <selector|json>     Extract data (CSS selector or JSON schema)")
	fmt.Println("  press <key>                Press key (enter, tab, escape, arrowdown...)")
	fmt.Println("  scroll                     Scroll page down")
	fmt.Println("  wait_url <substring>       Wait until URL contains substring")
	fmt.Println("  wait_element <text>        Wait until page contains text")
	fmt.Println("  wait_idle                  Wait for page network/DOM idle")
	fmt.Println("  wait_2fa <index>           Mark 2FA input field for replay")
	fmt.Println("  screenshot [label]         Take a screenshot")
	fmt.Println("  refresh                    Re-scan the UI map")
	fmt.Println("  done                       Finish and save flow")
	fmt.Println()
}

// RunManualMode lets the user control the browser by typing commands in the terminal.
func RunManualMode(page *rod.Page, config AgentConfig) error {
	parsed, _ := url.Parse(config.StartURL)
	hostname := parsed.Hostname()

	scanner := bufio.NewScanner(os.Stdin)
	var allSteps []ActionStep
	var pageSignatures []int
	finished := false

	for iter := 0; iter < MaxLoopIter && !finished; iter++ {
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

		PrintManualBanner(currentURL, count)
		fmt.Println(uiMap)
		PrintManualHelp()

		rescan := false
		for !finished && !rescan {
			fmt.Printf("%smanual> %s", ColorGreen, ColorReset)
			if !scanner.Scan() {
				finished = true
				break
			}

			step, isRefresh, err := ParseManualCommand(scanner.Text(), len(elements))
			if err != nil {
				msg := err.Error()
				if msg != "" {
					LogError("MANUAL", "%s", msg)
				}
				continue
			}

			if isRefresh {
				LogInfo("MANUAL", "Refreshing UI map...")
				rescan = true
				break
			}

			if strings.ToUpper(step.Action) == "DONE" {
				allSteps = append(allSteps, step)
				finished = true
				break
			}

			if step.Index > 0 && step.Index <= len(elements) {
				step.X = elements[step.Index-1].X
				step.Y = elements[step.Index-1].Y
			}

			config.LastScrapeResult = nil
			execErr := ExecuteStep(page, step, elements, config.Variables, &config)
			if execErr != nil {
				LogError("MANUAL", "Action failed: %v", execErr)
				fmt.Println("  (step not recorded — try again or use a different command)")
				continue
			}

			// Print scrape results to the terminal.
			if strings.ToUpper(step.Action) == "SCRAPE" && config.LastScrapeResult != nil {
				prettyJSON, err := json.MarshalIndent(config.LastScrapeResult, "  ", "  ")
				if err == nil {
					fmt.Printf("\n%s  ┌─ SCRAPE RESULT ──────────────────────────────────%s\n", ColorCyan, ColorReset)
					fmt.Printf("  %s\n", string(prettyJSON))
					fmt.Printf("%s  └──────────────────────────────────────────────────%s\n\n", ColorCyan, ColorReset)
				}
			}

			allSteps = append(allSteps, ActionStep{
				Action: step.Action, Index: step.Index, X: step.X, Y: step.Y, Value: step.Value,
			})
			LogSuccess("MANUAL", "Recorded step %d: %s", len(allSteps), FormatStep(step))

			if strings.ToUpper(step.Action) == "CLICK" || strings.ToUpper(step.Action) == "WAIT_URL" {
				rescan = true
			}
		}
	}

	if len(allSteps) == 0 {
		LogWarning("MANUAL", "No steps recorded. Nothing to save.")
		return nil
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
	LogSuccess("SUCCESS", "Objective '%s' completed in %d steps (Manual Mode)", config.Objective, len(allSteps))
	return nil
}
