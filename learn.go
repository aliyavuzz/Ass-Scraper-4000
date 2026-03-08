package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// ============================================================================
// LEARNING MODE
// ============================================================================

// JsLearnModeListeners is injected into every page to capture user interactions.
const JsLearnModeListeners = `
(function() {
    const PREFIX = '__AGENT_EVENT__';

    document.addEventListener('click', function(e) {
        var el = e.target;
        var rect = el.getBoundingClientRect();
        var cx = rect.left + rect.width / 2;
        var cy = rect.top + rect.height / 2;
        var text = (el.innerText || el.placeholder || el.title || el.getAttribute('aria-label') || '').trim();
        if (text.length > 60) text = text.substring(0, 60);
        var data = JSON.stringify({
            type: 'CLICK',
            x: Math.round(cx * 10) / 10,
            y: Math.round(cy * 10) / 10,
            tag: el.tagName.toLowerCase(),
            id: el.id || '',
            name: el.name || '',
            text: text
        });
        console.log(PREFIX + data);
    }, true);

    document.addEventListener('input', function(e) {
        var el = e.target;
        var tag = el.tagName.toLowerCase();
        if (tag !== 'input' && tag !== 'textarea') return;
        var rect = el.getBoundingClientRect();
        var cx = rect.left + rect.width / 2;
        var cy = rect.top + rect.height / 2;
        var data = JSON.stringify({
            type: 'TYPE',
            x: Math.round(cx * 10) / 10,
            y: Math.round(cy * 10) / 10,
            value: el.value,
            tag: tag,
            id: el.id || '',
            name: el.name || ''
        });
        console.log(PREFIX + data);
    }, true);

    document.addEventListener('change', function(e) {
        var el = e.target;
        if (el.tagName.toLowerCase() !== 'select') return;
        var rect = el.getBoundingClientRect();
        var cx = rect.left + rect.width / 2;
        var cy = rect.top + rect.height / 2;
        var selected = el.options[el.selectedIndex];
        var data = JSON.stringify({
            type: 'SELECT',
            x: Math.round(cx * 10) / 10,
            y: Math.round(cy * 10) / 10,
            value: selected ? selected.text : el.value,
            tag: 'select',
            id: el.id || '',
            name: el.name || ''
        });
        console.log(PREFIX + data);
    }, true);
})();
`

const learnEventPrefix = "__AGENT_EVENT__"

// RunLearnMode opens the browser and records user interactions via injected JS listeners.
func RunLearnMode(page *rod.Page, config AgentConfig) error {
	parsed, _ := url.Parse(config.StartURL)
	hostname := parsed.Hostname()

	// Inject JS listeners that survive page navigations.
	_, err := page.EvalOnNewDocument(JsLearnModeListeners)
	if err != nil {
		return fmt.Errorf("failed to inject learning listeners: %w", err)
	}

	// Navigate (this triggers the injected JS on the new document).
	if err := NavigateTo(page, config.StartURL); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	// Get initial page signature.
	_, _, initCount, _ := GetUIMap(page)
	pageSignatures := []int{initCount}

	// Channel for captured events.
	events := make(chan LearnEvent, 500)
	doneCh := make(chan struct{})

	// Goroutine: listen for console events from the injected JS.
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) bool {
		if e.Type == proto.RuntimeConsoleAPICalledTypeLog {
			for _, arg := range e.Args {
				text := arg.Value.Str()
				if strings.HasPrefix(text, learnEventPrefix) {
					payload := strings.TrimPrefix(text, learnEventPrefix)
					var evt LearnEvent
					if json.Unmarshal([]byte(payload), &evt) == nil {
						select {
						case events <- evt:
						default:
						}
					}
				}
			}
		}
		select {
		case <-doneCh:
			return true
		default:
			return false
		}
	})()

	// Goroutine: read terminal for "done" signal.
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if line == "done" {
				close(doneCh)
				return
			}
			if line == "screenshot" {
				TakeScreenshot(page, hostname+"_learn")
			}
		}
	}()

	fmt.Printf("\n%s══════════════════════════════════════════════════════════%s\n", ColorMagenta, ColorReset)
	fmt.Printf("%s  LEARNING MODE ACTIVE%s\n", ColorBold, ColorReset)
	fmt.Printf("  Interact with the browser. Every click, type, and\n")
	fmt.Printf("  select will be recorded automatically.\n")
	fmt.Printf("\n")
	fmt.Printf("  Type %sdone%s here when finished.\n", ColorGreen, ColorReset)
	fmt.Printf("  Type %sscreenshot%s to capture the current page.\n", ColorGreen, ColorReset)
	fmt.Printf("%s══════════════════════════════════════════════════════════%s\n\n", ColorMagenta, ColorReset)

	// Collect events with TYPE coalescing.
	var rawSteps []ActionStep
	var mu sync.Mutex
	lastTypeKey := ""

loop:
	for {
		select {
		case evt := <-events:
			mu.Lock()
			step := learnEventToStep(evt)

			if step.Action == "TYPE" {
				key := fmt.Sprintf("%.1f,%.1f", step.X, step.Y)
				if key == lastTypeKey && len(rawSteps) > 0 {
					rawSteps[len(rawSteps)-1] = step
					mu.Unlock()
					continue
				}
				lastTypeKey = key
			} else {
				lastTypeKey = ""
			}

			rawSteps = append(rawSteps, step)
			mu.Unlock()
			LogAction("LEARN", "Captured: %s", FormatStep(step))

		case <-doneCh:
			drainTimer := time.After(500 * time.Millisecond)
		drain:
			for {
				select {
				case evt := <-events:
					mu.Lock()
					step := learnEventToStep(evt)
					if step.Action == "TYPE" {
						key := fmt.Sprintf("%.1f,%.1f", step.X, step.Y)
						if key == lastTypeKey && len(rawSteps) > 0 {
							rawSteps[len(rawSteps)-1] = step
							mu.Unlock()
							continue
						}
						lastTypeKey = key
					} else {
						lastTypeKey = ""
					}
					rawSteps = append(rawSteps, step)
					mu.Unlock()
				case <-drainTimer:
					break drain
				}
			}
			break loop
		}
	}

	if len(rawSteps) == 0 {
		LogWarning("LEARN", "No interactions captured. Nothing to save.")
		return nil
	}

	rawSteps = append(rawSteps, ActionStep{Action: "DONE"})

	fmt.Printf("\n%s══════════════════════════════════════════════════════════%s\n", ColorCyan, ColorReset)
	fmt.Printf("  Captured %d steps. Now let's make inputs reusable.\n", len(rawSteps)-1)
	fmt.Printf("%s══════════════════════════════════════════════════════════%s\n\n", ColorCyan, ColorReset)

	templatedSteps, varNames := PromptVariableTemplating(rawSteps, config)

	flow := FlowFile{
		Site: hostname, Objective: config.Objective,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		PageSignatures: pageSignatures, Steps: templatedSteps,
		Variables: varNames,
	}
	if err := SaveFlow(flow); err != nil {
		return err
	}
	LogSuccess("SUCCESS", "Objective '%s' recorded in %d steps (Learning Mode)", config.Objective, len(rawSteps)-1)
	return nil
}

func learnEventToStep(evt LearnEvent) ActionStep {
	return ActionStep{
		Action: evt.Type,
		X:      evt.X,
		Y:      evt.Y,
		Value:  evt.Value,
	}
}

// PromptVariableTemplating asks the user which TYPE values should be turned into {{variables}}.
func PromptVariableTemplating(steps []ActionStep, config AgentConfig) ([]ActionStep, []string) {
	result := make([]ActionStep, len(steps))
	copy(result, steps)

	if len(config.Variables) > 0 {
		result = TemplateSteps(result, config.Variables)
		LogInfo("LEARN", "Auto-templated %d known variables", len(config.Variables))
	}

	scanner := bufio.NewScanner(os.Stdin)
	varMap := map[string]string{}

	for i, step := range result {
		if strings.ToUpper(step.Action) != "TYPE" {
			continue
		}
		if step.Value == "" || strings.Contains(step.Value, "{{") {
			continue
		}

		alreadyNamed := ""
		for vn, vv := range varMap {
			if vv == step.Value {
				alreadyNamed = vn
				break
			}
		}
		if alreadyNamed != "" {
			result[i].Value = "{{" + alreadyNamed + "}}"
			LogInfo("LEARN", "Auto-applied {{%s}} to step %d", alreadyNamed, i+1)
			continue
		}

		fmt.Printf("  Step %d: TYPE value=%q at (%.0f, %.0f)\n", i+1, step.Value, step.X, step.Y)
		fmt.Printf("  %sVariable name for this value?%s (leave empty to keep literal): ", ColorYellow, ColorReset)

		if !scanner.Scan() {
			break
		}
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			continue
		}

		varMap[name] = step.Value
		result[i].Value = "{{" + name + "}}"
		LogSuccess("LEARN", "Mapped %q -> {{%s}}", step.Value, name)

		for j := i + 1; j < len(result); j++ {
			if strings.ToUpper(result[j].Action) == "TYPE" && result[j].Value == step.Value {
				result[j].Value = "{{" + name + "}}"
			}
		}
	}

	names := ExtractVariableNames(result)
	return result, names
}
