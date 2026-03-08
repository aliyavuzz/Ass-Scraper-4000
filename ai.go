package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-rod/rod"
)

// ============================================================================
// UI MAPPING ENGINE
// ============================================================================

const jsExtractElements = `
    const selectors = 'input, button, select, textarea, a[href], [role="button"], [role="link"], [role="menuitem"], [role="tab"]';
    const els = document.querySelectorAll(selectors);
    const results = [];
    for (let i = 0; i < els.length; i++) {
        const el = els[i];
        const style = window.getComputedStyle(el);
        if (style.display === 'none' || style.visibility === 'hidden') continue;
        const rect = el.getBoundingClientRect();
        if (rect.width === 0 && rect.height === 0) continue;
        const cx = rect.left + rect.width / 2;
        const cy = rect.top + rect.height / 2;
        if (cx < 0 || cy < 0 || cx > %d || cy > %d) continue;
        let text = (el.innerText || el.placeholder || el.title || el.getAttribute('aria-label') || '').trim();
        if (text.length > 60) text = text.substring(0, 60);
        results.push({
            tag:   el.tagName.toLowerCase(),
            id:    el.id || '',
            name:  el.name || '',
            text:  text,
            value: el.value || '',
            type:  el.type || '',
            href:  el.href || '',
            x:     Math.round(cx * 10) / 10,
            y:     Math.round(cy * 10) / 10
        });
    }
    return JSON.stringify(results);
`

func GetUIMap(page *rod.Page) ([]UIElement, string, int, error) {
	pageInfo, err := page.Info()
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to get page info: %w", err)
	}
	currentURL := pageInfo.URL

	js := fmt.Sprintf(jsExtractElements, ViewportWidth, ViewportHeight)

	res, err := page.Eval("() => {" + js + "}")
	if err != nil {
		return nil, "", 0, fmt.Errorf("JS evaluation failed: %w", err)
	}

	var rawEls []RawElement
	if err := json.Unmarshal([]byte(res.Value.Str()), &rawEls); err != nil {
		return nil, "", 0, fmt.Errorf("failed to parse elements JSON: %w", err)
	}

	elements := make([]UIElement, 0, len(rawEls))
	for i, r := range rawEls {
		elements = append(elements, UIElement{
			Index: i + 1, Tag: r.Tag, ID: r.ID, Name: r.Name,
			Text: r.Text, Value: r.Value, Type: r.Type, Href: r.Href,
			X: r.X, Y: r.Y,
		})
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# UI Map — %s\n", currentURL))
	sb.WriteString(fmt.Sprintf("## Interactive Elements (%d total)\n\n", len(elements)))
	for _, el := range elements {
		line := fmt.Sprintf("[%d] %s", el.Index, strings.ToUpper(el.Tag))
		if el.Type != "" {
			line += fmt.Sprintf(" | type=%s", el.Type)
		}
		if el.ID != "" {
			line += fmt.Sprintf(" | id=%s", el.ID)
		}
		if el.Name != "" {
			line += fmt.Sprintf(" | name=%s", el.Name)
		}
		if el.Text != "" {
			line += fmt.Sprintf(" | text=\"%s\"", el.Text)
		}
		if el.Value != "" {
			line += fmt.Sprintf(" | value=\"%s\"", el.Value)
		}
		if el.Href != "" {
			line += fmt.Sprintf(" | href=\"%s\"", el.Href)
		}
		line += fmt.Sprintf(" | (%.0f, %.0f)", el.X, el.Y)
		sb.WriteString(line + "\n")
	}

	LogInfo("UI MAP", "Found %d elements on %s", len(elements), currentURL)
	return elements, sb.String(), len(elements), nil
}

// ============================================================================
// AI DECISION LAYER
// ============================================================================

func geminiURL(apiKey string) string {
	return "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=" + apiKey
}

func AskLLM(apiKey, uiMap, objective string, variables map[string]string, pageURL string) (LLMResponse, error) {
	varNames := make([]string, 0, len(variables))
	for k := range variables {
		varNames = append(varNames, k)
	}
	varList := strings.Join(varNames, ", ")

	systemPrompt := fmt.Sprintf(`You are a browser automation agent. You receive a structured list of interactive UI elements on a webpage and must decide what actions to take to achieve the given objective.

Rules:
- You can only interact with elements listed in the UI Map by their index number.
- Available actions: CLICK (click element at index), TYPE (clear and type value into element at index), SELECT (choose option by value in a select element), SCROLL (scroll page down), WAIT_URL (pause until URL contains the given value string), WAIT_ELEMENT (pause until page contains the given text string), SCREENSHOT (take a screenshot of current page), DONE (objective achieved).
- For TYPE actions involving sensitive variables like username or password, use the placeholder syntax {{variable_name}} as the value. Available variables: %s.
- Return a JSON object ONLY. No markdown, no explanation outside JSON.
- Batch all actions for the current page into a single response. Do not return actions for future pages.
- If the objective is already achieved on this page, return DONE.
- Use SCREENSHOT before DONE if you want to capture the final state.

Response format:
{
  "steps": [
    {"action": "TYPE", "index": 1, "value": "{{username}}"},
    {"action": "TYPE", "index": 2, "value": "{{password}}"},
    {"action": "CLICK", "index": 3},
    {"action": "WAIT_URL", "value": "/dashboard"}
  ],
  "goal_reached": false,
  "reasoning": "Filling login form and submitting."
}`, varList)

	userPrompt := fmt.Sprintf("Current page URL: %s\nObjective: %s\n\n%s", pageURL, objective, uiMap)
	fullPrompt := systemPrompt + "\n\n" + userPrompt

	var llmResp LLMResponse
	var lastErr error

	for attempt := 0; attempt <= MaxLLMRetries; attempt++ {
		prompt := fullPrompt
		if attempt > 0 {
			prompt += "\n\nYour previous response was not valid JSON. Return ONLY a raw JSON object."
			LogWarning("AI", "Retrying LLM call (attempt %d/%d)", attempt+1, MaxLLMRetries+1)
		}
		LogInfo("AI", "Sending UI map to Gemini (%d chars)", len(prompt))
		respText, err := callGeminiAPI(apiKey, prompt)
		if err != nil {
			lastErr = err
			continue
		}
		respText = cleanJSONResponse(respText)
		if err := json.Unmarshal([]byte(respText), &llmResp); err != nil {
			lastErr = fmt.Errorf("JSON parse error: %w (raw: %s)", err, Truncate(respText, 200))
			LogError("AI", "Failed to parse LLM response: %v", lastErr)
			continue
		}
		LogAI("AI", "Reasoning: %s", llmResp.Reasoning)
		LogAI("AI", "Steps: %d, GoalReached: %v", len(llmResp.Steps), llmResp.GoalReached)
		return llmResp, nil
	}
	return LLMResponse{}, fmt.Errorf("LLM failed after %d attempts: %w", MaxLLMRetries+1, lastErr)
}

func callGeminiAPI(apiKey, prompt string) (string, error) {
	reqBody := GeminiRequest{
		Contents:         []GeminiContent{{Parts: []GeminiPart{{Text: prompt}}}},
		GenerationConfig: GeminiGenConfig{Temperature: 0.1, ResponseMimeType: "application/json"},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	httpResp, err := http.Post(geminiURL(apiKey), "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()
	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API error %d: %s", httpResp.StatusCode, Truncate(string(respBytes), 300))
	}
	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal Gemini response: %w", err)
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty Gemini response")
	}
	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func cleanJSONResponse(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
