package engine

import (
	"github.com/go-rod/rod"
)

// ============================================================================
// CONSTANTS
// ============================================================================

const (
	ViewportWidth  = 1280
	ViewportHeight = 800
	MaxRetries     = 3
	MaxLLMRetries  = 2
	FlowDir        = "./flows"
	ScreenshotDir  = "./screenshots"
	StaleThreshold = 0.4
	MaxLoopIter    = 20
)

// ============================================================================
// DATA STRUCTS
// ============================================================================

type UIElement struct {
	Index int
	Tag   string
	ID    string
	Name  string
	Text  string
	Value string
	Type  string
	Href  string
	X     float64
	Y     float64
}

type ActionStep struct {
	Action string  `json:"action"`
	Index  int     `json:"index,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Value  string  `json:"value,omitempty"`
	Result any     `json:"result,omitempty"`
}

type RecipeResult struct {
	Success    bool           `json:"success"`
	Steps      int            `json:"steps_executed"`
	Data       map[string]any `json:"data,omitempty"`
	Screenshot string         `json:"screenshot,omitempty"`
	Error      string         `json:"error,omitempty"`
	Duration   string         `json:"duration"`
}

type LLMResponse struct {
	Steps       []ActionStep `json:"steps"`
	GoalReached bool         `json:"goal_reached"`
	Reasoning   string       `json:"reasoning"`
}

type FlowFile struct {
	Site           string       `json:"site"`
	Objective      string       `json:"objective"`
	RecordedAt     string       `json:"recorded_at"`
	PageSignatures []int        `json:"page_signatures"`
	Steps          []ActionStep `json:"steps"`
	Variables      []string     `json:"variables"`
}

type AgentConfig struct {
	StartURL   string
	Objective  string
	Variables  map[string]string
	GeminiKey  string
	Headless   bool
	Screenshot bool
	// OnStatus is called with log messages during execution (used by web UI).
	OnStatus func(msg string)
	// TwoFAChan delivers a 2FA code when a WAIT_2FA step is encountered.
	TwoFAChan chan string
	// LastScrapeResult holds the result of the most recent SCRAPE action.
	LastScrapeResult any
}

type Credential struct {
	Carrier       string
	Tenant        string
	AdminUserName string
	AdminPassword string
	AdminCode     string
	AdminUser     string
	URL           string
}

// LearnEvent is the JSON shape sent from the injected JS during learning mode.
type LearnEvent struct {
	Type  string  `json:"type"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Value string  `json:"value"`
	Tag   string  `json:"tag"`
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Text  string  `json:"text"`
}

type GeminiRequest struct {
	Contents         []GeminiContent `json:"contents"`
	GenerationConfig GeminiGenConfig `json:"generationConfig"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiGenConfig struct {
	Temperature      float64 `json:"temperature"`
	ResponseMimeType string  `json:"responseMimeType"`
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

type ErrorLevel int

const (
	RetryAction    ErrorLevel = iota
	RegenerateFlow
	Fatal
)

// CommonFlags holds parsed CLI flags shared by run/manual/learn commands.
type CommonFlags struct {
	Carrier    string
	Tenant     string
	DirectURL  string
	User       string
	Pass       string
	Objective  string
	CSVPath    string
	Headless   bool
	Screenshot bool
	APIKey     string // only used by "run" command
}

// RawElement is used internally for JSON deserialization of UI map extraction.
type RawElement struct {
	Tag   string  `json:"tag"`
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Text  string  `json:"text"`
	Value string  `json:"value"`
	Type  string  `json:"type"`
	Href  string  `json:"href"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

// BrowserInstance groups a browser and its primary page for convenience.
type BrowserInstance struct {
	Browser *rod.Browser
	Page    *rod.Page
}
