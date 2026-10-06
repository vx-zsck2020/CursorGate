package main

import (
	"os"
	"strings"
	"testing"
)

func TestFixedWindowSize(t *testing.T) {
	if windowWidth != 875 {
		t.Fatalf("windowWidth = %d, want 875", windowWidth)
	}
	if windowHeight != 525 {
		t.Fatalf("windowHeight = %d, want 525", windowHeight)
	}
	if appTitle != "CursorGate - Cursor自定义API助手" {
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

	winSrc, err := os.ReadFile("window_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(winSrc), "func lockClientSize()") {
		t.Fatal("window_windows.go missing lockClientSize")
	}
	if !strings.Contains(string(winSrc), "AdjustWindowRectEx") {
		t.Fatal("window_windows.go should compensate title-bar chrome")
	}
	if !strings.Contains(string(winSrc), "spiGetWorkArea") {
		t.Fatal("window_windows.go should clamp to work area")
	}
	if strings.Contains(text, "MinWidth:") || strings.Contains(text, "MaxWidth:") {
		t.Fatal("Min/Max must stay unset so chrome compensation can grow the outer frame")
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
}
