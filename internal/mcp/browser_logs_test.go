package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
	"github.com/geodro/lerd/internal/dumps"
)

func browserEventsText(t *testing.T, args map[string]any) map[string]any {
	t.Helper()
	res, _ := execBrowserEvents(args)
	m := res.(map[string]any)
	text := m["content"].([]map[string]any)[0]["text"].(string)
	if m["isError"] == true {
		return map[string]any{"error": text}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("decoding %s: %v", text, err)
	}
	return out
}

func TestBrowserEvents_ExplainsAnEmptyAnswer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	stubRoundTrip(t, "[]")
	off := false
	if err := config.AddSite(config.Site{Name: "shop", Domains: []string{"shop.test"}, Path: t.TempDir(), BrowserLogs: &config.BrowserLogs{Enabled: &off}}); err != nil {
		t.Fatal(err)
	}

	siteOff := browserEventsText(t, map[string]any{"site": "shop"})
	if siteOff["site_enabled"] != false || !strings.Contains(siteOff["hint"].(string), "browser_toggle") {
		t.Fatalf("site off = %v", siteOff)
	}
	debugOff := browserEventsText(t, map[string]any{})
	if debugOff["debug_enabled"] != false || !strings.Contains(debugOff["hint"].(string), "dumps_toggle") {
		t.Fatalf("debug off = %v", debugOff)
	}
	cfg, _ := config.LoadGlobal()
	cfg.SetDumpsEnabled(true)
	if err := config.SaveGlobal(cfg); err != nil {
		t.Fatal(err)
	}
	on := browserEventsText(t, map[string]any{})
	if !strings.Contains(on["hint"].(string), "Load or reload a page") {
		t.Fatalf("on = %v", on)
	}
}

func TestBrowserToggle_NeedsASite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	res, _ := execBrowserLogsToggle(map[string]any{"enable": true})
	m := res.(map[string]any)
	if m["isError"] != true || !strings.Contains(m["content"].([]map[string]any)[0]["text"].(string), `"site"`) {
		t.Fatalf("got %v", m)
	}
}

func TestBrowserEvents_RefusesAnUnknownType(t *testing.T) {
	stubRoundTrip(t, "[]")
	got := browserEventsText(t, map[string]any{"types": []any{"errors"}})
	if !strings.Contains(got["error"].(string), `"errors"`) {
		t.Fatalf("got %v", got)
	}
}

// A types filter that empties the answer is the clean result, not missing
// capture, so the hint counts the page views that did arrive.
func TestBrowserEvents_FilteredEmptyCountsThePageViews(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	data, _ := json.Marshal(map[string]any{"type": "navigation", "message": "https://shop.test/"})
	events, _ := json.Marshal([]dumps.Event{
		{ID: "1", Kind: dumps.KindBrowser, Data: data, Ctx: dumps.Context{RID: "p1", Request: "https://shop.test/"}},
		{ID: "2", Kind: dumps.KindBrowser, Data: data, Ctx: dumps.Context{RID: "p2", Request: "https://shop.test/"}},
	})
	stubRoundTrip(t, string(events))
	cfg, _ := config.LoadGlobal()
	cfg.SetDumpsEnabled(true)
	if err := config.SaveGlobal(cfg); err != nil {
		t.Fatal(err)
	}

	out := browserEventsText(t, map[string]any{"types": []any{"error"}})
	if hint, _ := out["hint"].(string); !strings.Contains(hint, "2 page views") {
		t.Fatalf("hint = %q, want it to count the 2 page views recorded", hint)
	}
}

// limit cuts events before types narrow them, so a page's error may be the part
// that was cut; the hint must not call the pages clean then.
func TestBrowserEvents_FilteredEmptyUnderALimitIsNotClean(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	data, _ := json.Marshal(map[string]any{"type": "console", "level": "warn", "message": "late warning"})
	events, _ := json.Marshal([]dumps.Event{{ID: "1", Kind: dumps.KindBrowser, Data: data, Ctx: dumps.Context{RID: "p1", Request: "https://shop.test/"}}})
	stubRoundTrip(t, string(events))
	cfg, _ := config.LoadGlobal()
	cfg.SetDumpsEnabled(true)
	if err := config.SaveGlobal(cfg); err != nil {
		t.Fatal(err)
	}

	out := browserEventsText(t, map[string]any{"types": []any{"error"}, "limit": float64(1)})
	hint, _ := out["hint"].(string)
	if strings.Contains(hint, "reported none") || !strings.Contains(hint, "limit") {
		t.Fatalf("hint = %q, want it to blame the limit rather than call the page clean", hint)
	}
}
