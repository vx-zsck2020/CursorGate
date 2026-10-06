//go:build windows

package main

import "testing"

func TestChromeForCaptionWindow(t *testing.T) {
	const (
		wsCaption      = 0x00C00000
		wsSysMenu      = 0x00080000
		wsMinimizeBox  = 0x00020000
		wsVisible      = 0x10000000
		wsClipSiblings = 0x04000000
		wsClipChildren = 0x02000000
	)
	style := uintptr(wsCaption | wsSysMenu | wsMinimizeBox | wsVisible | wsClipSiblings | wsClipChildren)
	w, h := chromeForStyle(style, 0)
	if w <= 0 || h <= 0 {
		t.Fatalf("chrome compensation empty: %dx%d", w, h)
	}
	if w > 40 || h > 80 {
		t.Fatalf("chrome compensation implausible: %dx%d", w, h)
	}
}

func TestClampClientFitsWorkArea(t *testing.T) {
	w, h := clampClient(875, 525, 1920, 1040, 16, 39)
	if w != 875 || h != 525 {
		t.Fatalf("normal screen got %dx%d", w, h)
	}
	w, h = clampClient(875, 525, 800, 500, 16, 39)
	if w > 800-16-16 {
		t.Fatalf("small width not clamped: %d", w)
	}
	if h > 500-39-16 {
		t.Fatalf("small height not clamped: %d", h)
	}
	if w < minClientWidth {
		t.Fatalf("width below floor: %d", w)
	}
}
