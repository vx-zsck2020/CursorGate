//go:build windows

package native

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	modUser32                    = syscall.NewLazyDLL("user32.dll")
	procFindWindowW              = modUser32.NewProc("FindWindowW")
	procGetWindowLongW           = modUser32.NewProc("GetWindowLongW")
	procGetWindowRect            = modUser32.NewProc("GetWindowRect")
	procGetClientRect            = modUser32.NewProc("GetClientRect")
	procAdjustWindowRectEx       = modUser32.NewProc("AdjustWindowRectEx")
	procSetWindowPos             = modUser32.NewProc("SetWindowPos")
	procEnumWindows              = modUser32.NewProc("EnumWindows")
	procIsWindowVisible          = modUser32.NewProc("IsWindowVisible")
	procGetWindowThreadProcessId = modUser32.NewProc("GetWindowThreadProcessId")
	procSystemParametersInfoW    = modUser32.NewProc("SystemParametersInfoW")
	procGetCurrentProcessId      = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentProcessId")
)

const (
	gwlStyle        int32 = -16
	gwlExStyle      int32 = -20
	swpNoMove             = 0x0002
	swpNoZOrder           = 0x0004
	swpNoActivate         = 0x0010
	spiGetWorkArea        = 0x0030
	minClientWidth        = 640
	minClientHeight       = 400
	workAreaMargin        = 16
)

type winRect struct {
	Left, Top, Right, Bottom int32
}

var (
	windowTitle  string
	windowWidth  = 875
	windowHeight = 525
)

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func findAppWindow() uintptr {
	hwnd, _, _ := procFindWindowW.Call(
		uintptr(unsafe.Pointer(utf16Ptr("wailsWindow"))),
		uintptr(unsafe.Pointer(utf16Ptr(windowTitle))),
	)
	if hwnd != 0 {
		return hwnd
	}
	pid, _, _ := procGetCurrentProcessId.Call()
	var found uintptr
	cb := syscall.NewCallback(func(h uintptr, _ uintptr) uintptr {
		var wpid uint32
		procGetWindowThreadProcessId.Call(h, uintptr(unsafe.Pointer(&wpid)))
		if uint32(pid) != wpid {
			return 1
		}
		vis, _, _ := procIsWindowVisible.Call(h)
		if vis == 0 {
			return 1
		}
		var r winRect
		procGetWindowRect.Call(h, uintptr(unsafe.Pointer(&r)))
		if r.Right-r.Left < 200 || r.Bottom-r.Top < 200 {
			return 1
		}
		found = h
		return 0
	})
	procEnumWindows.Call(cb, 0)
	return found
}

func nIndex(v int32) uintptr {
	return uintptr(uint32(v))
}

func windowStyle(hwnd uintptr) (style, exStyle uintptr) {
	style, _, _ = procGetWindowLongW.Call(hwnd, nIndex(gwlStyle))
	exStyle, _, _ = procGetWindowLongW.Call(hwnd, nIndex(gwlExStyle))
	return
}

func measureWindow(hwnd uintptr) (outerW, outerH, clientW, clientH int) {
	var outer, client winRect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&outer)))
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	outerW = int(outer.Right - outer.Left)
	outerH = int(outer.Bottom - outer.Top)
	clientW = int(client.Right - client.Left)
	clientH = int(client.Bottom - client.Top)
	return
}

func chromeForStyle(style, exStyle uintptr) (chromeW, chromeH int) {
	r := winRect{Right: int32(windowWidth), Bottom: int32(windowHeight)}
	ok, _, _ := procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&r)), style, 0, exStyle)
	if ok == 0 {
		return 16, 39
	}
	return int(r.Right-r.Left) - windowWidth, int(r.Bottom-r.Top) - windowHeight
}

func workArea() (w, h int) {
	var r winRect
	ok, _, _ := procSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0)
	if ok == 0 {
		return 1920, 1080
	}
	return int(r.Right - r.Left), int(r.Bottom - r.Top)
}

func clampClient(wantW, wantH, workW, workH, chromeW, chromeH int) (int, int) {
	maxW := workW - chromeW - workAreaMargin
	maxH := workH - chromeH - workAreaMargin
	if maxW < minClientWidth {
		maxW = minClientWidth
	}
	if maxH < minClientHeight {
		maxH = minClientHeight
	}
	if wantW > maxW {
		wantW = maxW
	}
	if wantH > maxH {
		wantH = maxH
	}
	if wantW < minClientWidth {
		wantW = minClientWidth
	}
	if wantH < minClientHeight {
		wantH = minClientHeight
	}
	return wantW, wantH
}

func applyOuterSize(hwnd uintptr, outerW, outerH int) {
	procSetWindowPos.Call(
		hwnd,
		0,
		0,
		0,
		uintptr(outerW),
		uintptr(outerH),
		swpNoMove|swpNoZOrder|swpNoActivate,
	)
}

func LockClientSize(title string, wantW, wantH int) {
	windowTitle = title
	if wantW > 0 {
		windowWidth = wantW
	}
	if wantH > 0 {
		windowHeight = wantH
	}
	hwnd := findAppWindow()
	if hwnd == 0 {
		time.Sleep(80 * time.Millisecond)
		hwnd = findAppWindow()
	}
	if hwnd == 0 {
		return
	}
	style, exStyle := windowStyle(hwnd)
	chromeW, chromeH := chromeForStyle(style, exStyle)
	if chromeW < 0 {
		chromeW = 0
	}
	if chromeH < 0 {
		chromeH = 0
	}
	workW, workH := workArea()
	clientW, clientH := clampClient(windowWidth, windowHeight, workW, workH, chromeW, chromeH)
	applyOuterSize(hwnd, clientW+chromeW, clientH+chromeH)

	_, _, gotW, gotH := measureWindow(hwnd)
	if gotW == clientW && gotH == clientH {
		return
	}
	outerW, outerH, _, _ := measureWindow(hwnd)
	applyOuterSize(hwnd, outerW+(clientW-gotW), outerH+(clientH-gotH))
}
