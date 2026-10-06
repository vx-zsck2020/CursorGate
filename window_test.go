package main

import (
	"os"
	"strings"
	"testing"

	"CursorGate/internal/update"
)

func TestFixedWindowSize(t *testing.T) {
	if windowWidth != 875 {
		t.Fatalf("windowWidth = %d, want 875", windowWidth)
	}
	if windowHeight != 525 {
		t.Fatalf("windowHeight = %d, want 525", windowHeight)
	}
	if appTitle != "CursorGate - Cursor自定义API助手 v"+update.AppVersion {
		t.Fatalf("appTitle = %q", appTitle)
	}

	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	for _, want := range []string{
		"DisableResize: true",
		"DisablePinchZoom:     true",
		"OnDomReady:       app.domReady",
		"OnBeforeClose:    app.beforeClose",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("main.go missing %q", want)
		}
	}
	if strings.Contains(text, "MinWidth:") || strings.Contains(text, "MaxWidth:") {
		t.Fatal("Min/Max must stay unset so chrome compensation can grow the outer frame")
	}

	winSrc, err := os.ReadFile("internal/native/window_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(winSrc), "func LockClientSize(") {
		t.Fatal("internal/native/window_windows.go missing LockClientSize")
	}
	if !strings.Contains(string(winSrc), "AdjustWindowRectEx") {
		t.Fatal("window compensation should use AdjustWindowRectEx")
	}
	if !strings.Contains(string(winSrc), "spiGetWorkArea") {
		t.Fatal("window should clamp to work area")
	}

	man, err := os.ReadFile("build/windows/wails.exe.manifest")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(man), `level="requireAdministrator"`) {
		t.Fatal("manifest must request administrator")
	}
	if !strings.Contains(string(man), "permonitorv2") {
		t.Fatal("manifest must keep per-monitor DPI")
	}

	ui, err := os.ReadFile("frontend/src/App.tsx")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ui), "127.0.0.1:8899") {
		t.Fatal("frontend placeholder must not use a local API port")
	}
	if !strings.Contains(string(ui), "https://api.example.com/v1") {
		t.Fatal("frontend should use a neutral example Base URL")
	}

	gi, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gi), "config.json") {
		t.Fatal(".gitignore must exclude user config.json")
	}
}
