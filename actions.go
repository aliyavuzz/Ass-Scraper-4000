package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// ============================================================================
// EXECUTION ENGINE
// ============================================================================

func ExecuteStep(page *rod.Page, step ActionStep, elements []UIElement, variables map[string]string, optCfg ...*AgentConfig) error {
	x, y := step.X, step.Y
	if elements != nil && step.Index > 0 && step.Index <= len(elements) {
		x = elements[step.Index-1].X
		y = elements[step.Index-1].Y
	}
	resolvedValue := SubstituteVariables(step.Value, variables)

	switch strings.ToUpper(step.Action) {
	case "CLICK":
		LogAction("ACTION", "CLICK at (%.0f, %.0f)", x, y)
		return executeClick(page, x, y)
	case "TYPE":
		LogAction("ACTION", "TYPE \"%s\" at (%.0f, %.0f)", MaskSensitive(resolvedValue, variables), x, y)
		return executeType(page, x, y, resolvedValue)
	case "SELECT":
		LogAction("ACTION", "SELECT \"%s\" at (%.0f, %.0f)", step.Value, x, y)
		return executeSelect(page, x, y, resolvedValue)
	case "SCROLL":
		LogAction("ACTION", "SCROLL down")
		return executeScroll(page)
	case "WAIT_URL":
		LogAction("ACTION", "WAIT_URL containing \"%s\"", step.Value)
		return executeWaitURL(page, step.Value)
	case "WAIT_ELEMENT":
		LogAction("ACTION", "WAIT_ELEMENT containing \"%s\"", step.Value)
		return executeWaitElement(page, step.Value)
	case "SCREENSHOT":
		LogAction("ACTION", "SCREENSHOT")
		_, err := TakeScreenshot(page, step.Value)
		return err
	case "WAIT_2FA":
		LogAction("ACTION", "WAIT_2FA — waiting for 2FA code at (%.0f, %.0f)", x, y)
		var cfg *AgentConfig
		if len(optCfg) > 0 {
			cfg = optCfg[0]
		}
		if cfg != nil && cfg.OnStatus != nil {
			cfg.OnStatus("__2FA_NEEDED__")
		}
		var code string
		if cfg != nil && cfg.TwoFAChan != nil {
			code = <-cfg.TwoFAChan
		} else {
			fmt.Print("  Enter 2FA code: ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				code = strings.TrimSpace(scanner.Text())
			}
		}
		if code == "" {
			return fmt.Errorf("no 2FA code provided")
		}
		LogAction("ACTION", "Typing 2FA code at (%.0f, %.0f)", x, y)
		return executeType(page, x, y, code)
	case "HOVER":
		LogAction("ACTION", "HOVER at (%.0f, %.0f)", x, y)
		return executeHover(page, x, y)
	case "UPLOAD":
		LogAction("ACTION", "UPLOAD \"%s\" at (%.0f, %.0f)", resolvedValue, x, y)
		return executeUpload(page, x, y, resolvedValue)
	case "SCRAPE":
		LogAction("ACTION", "SCRAPE schema: %s", step.Value)
		result, err := executeScrape(page, step.Value)
		if err != nil {
			return err
		}
		step.Result = result
		// Store result in config so callers can retrieve it.
		if len(optCfg) > 0 && optCfg[0] != nil {
			optCfg[0].LastScrapeResult = result
		}
		return nil
	case "WAIT_IDLE":
		LogAction("ACTION", "WAIT_IDLE — waiting for page idle")
		return executeWaitIdle(page)
	case "PRESS_KEY":
		LogAction("ACTION", "PRESS_KEY \"%s\"", step.Value)
		return executePressKey(page, step.Value)
	case "DONE":
		LogSuccess("ACTION", "DONE — Objective achieved")
		return nil
	default:
		return fmt.Errorf("unknown action: %s", step.Action)
	}
}

func executeClick(page *rod.Page, x, y float64) error {
	// Mouse jitter: ±2px for stealth.
	jx := x + float64(rand.Intn(5)) - 2
	jy := y + float64(rand.Intn(5)) - 2
	el, err := page.ElementFromPoint(int(jx), int(jy))
	if err != nil {
		// Fallback to exact coordinates.
		el, err = page.ElementFromPoint(int(x), int(y))
		if err != nil {
			return fmt.Errorf("ElementFromPoint(%d, %d) failed: %w", int(x), int(y), err)
		}
	}
	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click failed: %w", err)
	}
	time.Sleep(500 * time.Millisecond)
	WaitForPageReady(page)
	return nil
}

func executeType(page *rod.Page, x, y float64, value string) error {
	el, err := page.ElementFromPoint(int(x), int(y))
	if err != nil {
		return fmt.Errorf("ElementFromPoint(%d, %d) failed: %w", int(x), int(y), err)
	}
	_ = el.SelectAllText()
	if err := el.Input(value); err != nil {
		return fmt.Errorf("input failed: %w", err)
	}
	time.Sleep(300 * time.Millisecond)
	return nil
}

func executeSelect(page *rod.Page, x, y float64, value string) error {
	el, err := page.ElementFromPoint(int(x), int(y))
	if err != nil {
		return fmt.Errorf("ElementFromPoint(%d, %d) failed: %w", int(x), int(y), err)
	}
	if err := el.Select([]string{value}, true, rod.SelectorTypeText); err != nil {
		return fmt.Errorf("select failed: %w", err)
	}
	time.Sleep(300 * time.Millisecond)
	return nil
}

func executeScroll(page *rod.Page) error {
	_, err := page.Evaluate(rod.Eval("() => { window.scrollBy(0, 400); }"))
	if err != nil {
		return fmt.Errorf("scroll failed: %w", err)
	}
	time.Sleep(500 * time.Millisecond)
	return nil
}

func executeWaitURL(page *rod.Page, expected string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Try event-based approach first for 2s.
	eventCh := make(chan struct{}, 1)
	go func() {
		page.EachEvent(func(e *proto.PageFrameNavigated) bool {
			info, err := page.Info()
			if err == nil && strings.Contains(info.URL, expected) {
				select {
				case eventCh <- struct{}{}:
				default:
				}
				return true
			}
			return false
		})()
	}()

	// Also poll for SPA hash changes.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			info, _ := page.Info()
			cur := ""
			if info != nil {
				cur = info.URL
			}
			return fmt.Errorf("WAIT_URL timeout: URL %q does not contain %q", cur, expected)
		case <-eventCh:
			LogSuccess("ACTION", "URL now contains \"%s\"", expected)
			// Wait for SPA content to load.
			_ = page.WaitIdle(2 * time.Second)
			return nil
		case <-ticker.C:
			info, err := page.Info()
			if err != nil {
				continue
			}
			if strings.Contains(info.URL, expected) {
				LogSuccess("ACTION", "URL now contains \"%s\"", expected)
				_ = page.WaitIdle(2 * time.Second)
				return nil
			}
		}
	}
}

func executeWaitElement(page *rod.Page, expectedText string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("WAIT_ELEMENT timeout: text %q not found on page", expectedText)
		case <-ticker.C:
			js := fmt.Sprintf(`document.body.innerText.includes(%q)`, expectedText)
			res, err := page.Eval(js)
			if err != nil {
				continue
			}
			if res.Value.Bool() {
				LogSuccess("ACTION", "Found element containing \"%s\"", expectedText)
				return nil
			}
		}
	}
}

func executeHover(page *rod.Page, x, y float64) error {
	el, err := page.ElementFromPoint(int(x), int(y))
	if err != nil {
		return fmt.Errorf("ElementFromPoint(%d, %d) failed: %w", int(x), int(y), err)
	}
	if err := el.Hover(); err != nil {
		return fmt.Errorf("hover failed: %w", err)
	}
	time.Sleep(300 * time.Millisecond)
	return nil
}

func executeUpload(page *rod.Page, x, y float64, filePath string) error {
	el, err := page.ElementFromPoint(int(x), int(y))
	if err != nil {
		return fmt.Errorf("ElementFromPoint(%d, %d) failed: %w", int(x), int(y), err)
	}
	if err := el.SetFiles([]string{filePath}); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	time.Sleep(500 * time.Millisecond)
	return nil
}

func executeScrape(page *rod.Page, schema string) (any, error) {
	schema = strings.TrimSpace(schema)

	// Case 1: JSON object {"key": "selector", ...} or {"key": [".selector"]}
	if strings.HasPrefix(schema, "{") {
		var selectorMap map[string]any
		if err := json.Unmarshal([]byte(schema), &selectorMap); err != nil {
			return nil, fmt.Errorf("SCRAPE: invalid JSON schema: %w", err)
		}
		result := map[string]any{}
		for key, selRaw := range selectorMap {
			switch sel := selRaw.(type) {
			case string:
				text, err := scrapeText(page, sel)
				if err != nil {
					result[key] = nil
				} else {
					result[key] = text
				}
			case []any:
				if len(sel) > 0 {
					if s, ok := sel[0].(string); ok {
						texts, err := scrapeAllTexts(page, s)
						if err != nil {
							result[key] = []string{}
						} else {
							result[key] = texts
						}
					}
				}
			}
		}
		LogSuccess("ACTION", "SCRAPE: extracted %d fields", len(result))
		return result, nil
	}

	// Case 2: Plain CSS selector string.
	text, err := scrapeText(page, schema)
	if err != nil {
		return nil, fmt.Errorf("SCRAPE: selector %q failed: %w", schema, err)
	}
	LogSuccess("ACTION", "SCRAPE: got text (%d chars)", len(text))
	return text, nil
}

func scrapeText(page *rod.Page, selector string) (string, error) {
	el, err := page.Element(selector)
	if err != nil {
		return "", err
	}
	text, err := el.Text()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func scrapeAllTexts(page *rod.Page, selector string) ([]string, error) {
	els, err := page.Elements(selector)
	if err != nil {
		return nil, err
	}
	var texts []string
	for _, el := range els {
		t, err := el.Text()
		if err != nil {
			continue
		}
		texts = append(texts, strings.TrimSpace(t))
	}
	return texts, nil
}

func executeWaitIdle(page *rod.Page) error {
	// Phase 1: Network idle.
	if err := page.WaitIdle(2 * time.Second); err != nil {
		// Non-fatal: proceed to DOM stability check.
	}

	// Phase 2: DOM stability — two snapshots 500ms apart.
	var size1, size2 int
	res, err := page.Eval(`document.body.innerHTML.length`)
	if err == nil {
		size1 = res.Value.Int()
	}

	deadline := time.After(13 * time.Second)
	backoff := 500 * time.Millisecond
	for {
		select {
		case <-deadline:
			LogWarning("ACTION", "WAIT_IDLE: timeout, proceeding anyway")
			return nil
		default:
		}
		time.Sleep(backoff)
		res, err = page.Eval(`document.body.innerHTML.length`)
		if err == nil {
			size2 = res.Value.Int()
		}
		if size1 == size2 && size1 > 0 {
			LogSuccess("ACTION", "Page is idle (DOM stable at %d chars)", size1)
			return nil
		}
		size1 = size2
		if backoff < 2*time.Second {
			backoff = backoff * 3 / 2
		}
	}
}

// keyNameMap maps common key names to rod input.Key values.
var keyNameMap = map[string]input.Key{
	"enter": input.Enter, "tab": input.Tab, "escape": input.Escape, "esc": input.Escape,
	"backspace": input.Backspace, "delete": input.Delete, "space": input.Space,
	"arrowup": input.ArrowUp, "arrowdown": input.ArrowDown,
	"arrowleft": input.ArrowLeft, "arrowright": input.ArrowRight,
	"up": input.ArrowUp, "down": input.ArrowDown, "left": input.ArrowLeft, "right": input.ArrowRight,
	"home": input.Home, "end": input.End, "pageup": input.PageUp, "pagedown": input.PageDown,
	"f1": input.F1, "f2": input.F2, "f3": input.F3, "f4": input.F4,
	"f5": input.F5, "f6": input.F6, "f7": input.F7, "f8": input.F8,
	"f9": input.F9, "f10": input.F10, "f11": input.F11, "f12": input.F12,
}

func executePressKey(page *rod.Page, keyName string) error {
	name := strings.ToLower(strings.TrimSpace(keyName))
	if k, ok := keyNameMap[name]; ok {
		return page.Keyboard.Type(k)
	}
	if len(keyName) == 1 {
		return page.Keyboard.Type(input.Key(keyName[0]))
	}
	return fmt.Errorf("unknown key: %q (supported: enter, tab, escape, backspace, delete, space, arrowup/down/left/right, f1-f12, or single char)", keyName)
}

// ============================================================================
// VARIABLE HELPERS
// ============================================================================

func SubstituteVariables(value string, variables map[string]string) string {
	result := value
	for k, v := range variables {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}

func ReverseSubstituteVariables(value string, variables map[string]string) string {
	result := value
	for k, v := range variables {
		if v != "" {
			result = strings.ReplaceAll(result, v, "{{"+k+"}}")
		}
	}
	return result
}

func MaskSensitive(value string, variables map[string]string) string {
	result := value
	for k, v := range variables {
		if v != "" && strings.Contains(result, v) {
			result = strings.ReplaceAll(result, v, "{{"+k+"}}")
		}
	}
	return result
}
