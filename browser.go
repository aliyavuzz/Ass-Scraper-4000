package engine

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/go-rod/stealth"
)

// ============================================================================
// USER-AGENT POOL (for stealth rotation)
// ============================================================================

var userAgentPool = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
}

// ============================================================================
// BROWSER ENGINE
// ============================================================================

func InitBrowser(headless bool) (*rod.Browser, *rod.Page, error) {
	// Random viewport jitter (±20px).
	jitterW := ViewportWidth + rand.Intn(41) - 20
	jitterH := ViewportHeight + rand.Intn(41) - 20

	LogInfo("BROWSER", "Launching browser at %dx%d (headless=%v, stealth=on)", jitterW, jitterH, headless)

	u, err := launcher.New().Leakless(false).Headless(headless).Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	// Use go-rod/stealth for anti-bot fingerprint evasion.
	page, err := stealth.Page(browser)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stealth page: %w", err)
	}

	// Random User-Agent rotation.
	ua := userAgentPool[rand.Intn(len(userAgentPool))]
	_ = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{UserAgent: ua})

	err = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  jitterW,
		Height: jitterH,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set viewport: %w", err)
	}

	LogSuccess("BROWSER", "Browser launched successfully (stealth mode)")
	return browser, page, nil
}

func WaitForPageReady(page *rod.Page) {
	_ = page.WaitLoad()
	_ = page.WaitIdle(1 * time.Second)
}

func NavigateTo(page *rod.Page, targetURL string) error {
	LogInfo("BROWSER", "Navigating to %s", targetURL)
	if err := page.Navigate(targetURL); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}
	WaitForPageReady(page)
	return nil
}

// ============================================================================
// SCREENSHOT ENGINE
// ============================================================================

func TakeScreenshot(page *rod.Page, label string) (string, error) {
	if err := os.MkdirAll(ScreenshotDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create screenshot dir: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s.png", SanitizeFilename(label), timestamp)
	path := filepath.Join(ScreenshotDir, filename)

	data, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	})
	if err != nil {
		return "", fmt.Errorf("screenshot capture failed: %w", err)
	}

	if err := utils.OutputFile(path, data); err != nil {
		return "", fmt.Errorf("failed to save screenshot: %w", err)
	}

	LogSuccess("SCREEN", "Screenshot saved: %s", path)
	return path, nil
}
