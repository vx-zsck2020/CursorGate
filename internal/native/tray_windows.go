//go:build windows

package native

import (
	"syscall"
	"unsafe"
)

var (
	modShell32            = syscall.NewLazyDLL("shell32.dll")
	procShellNotifyIconW  = modShell32.NewProc("Shell_NotifyIconW")
	procExtractIconExW    = modShell32.NewProc("ExtractIconExW")
	procCreatePopupMenu   = modUser32.NewProc("CreatePopupMenu")
	procAppendMenuW       = modUser32.NewProc("AppendMenuW")
	procTrackPopupMenu    = modUser32.NewProc("TrackPopupMenu")
	procDestroyMenu       = modUser32.NewProc("DestroyMenu")
	procSetForegroundWnd  = modUser32.NewProc("SetForegroundWindow")
	procGetCursorPos      = modUser32.NewProc("GetCursorPos")
	procDestroyIcon       = modUser32.NewProc("DestroyIcon")
	procSetWindowLongPtrW = modUser32.NewProc("SetWindowLongPtrW")
	procCallWindowProcW   = modUser32.NewProc("CallWindowProcW")
	procGetModuleFileName = syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleFileNameW")
)

const (
	nimAdd               = 0x00000000
	nimDelete            = 0x00000002
	nifMessage           = 0x00000001
	nifIcon              = 0x00000002
	nifTip               = 0x00000004
	wmApp                = 0x8000
	wmTray               = wmApp + 1
	wmRButtonUp          = 0x0205
	wmLButtonUp          = 0x0202
	wmLButtonDbl         = 0x0203
	wmDestroy            = 0x0002
	wmCommand            = 0x0111
	gwlpWndProc    int32 = -4
	mfString             = 0x00000000
	tpmRightButton       = 0x0002
	trayOpenID           = 0xCC01
	trayExitID           = 0xCC02
)

type notifyIconData struct {
	CbSize           uint32
	Wnd              uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UTimeoutVersion  uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GUIDItem         [16]byte
	HBalloonIcon     uintptr
}

type point struct {
	X, Y int32
}

type TrayHooks struct {
	Show func()
	Quit func()
}

var (
	trayHWND     uintptr
	trayIcon     uintptr
	trayPrevProc uintptr
	trayNID      notifyIconData
	trayReady    bool
	trayHooks    TrayHooks
)

func StartTray(title string, hooks TrayHooks) {
	windowTitle = title
	trayHooks = hooks
	hwnd := findAppWindow()
	if hwnd == 0 {
		return
	}
	icon := loadAppIcon()
	nid := notifyIconData{
		Wnd:              hwnd,
		UID:              1,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: wmTray,
		HIcon:            icon,
	}
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	copyUTF16(nid.SzTip[:], title)
	ok, _, _ := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	if ok == 0 {
		return
	}
	cb := syscall.NewCallback(trayWndProc)
	prev, _, _ := procSetWindowLongPtrW.Call(hwnd, nIndex(gwlpWndProc), cb)
	trayHWND = hwnd
	trayIcon = icon
	trayPrevProc = prev
	trayNID = nid
	trayReady = true
}

func DestroyTray() {
	if !trayReady {
		return
	}
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&trayNID)))
	if trayPrevProc != 0 && trayHWND != 0 {
		procSetWindowLongPtrW.Call(trayHWND, nIndex(gwlpWndProc), trayPrevProc)
	}
	if trayIcon != 0 {
		procDestroyIcon.Call(trayIcon)
	}
	trayReady = false
	trayHWND = 0
	trayPrevProc = 0
	trayIcon = 0
}

func trayWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmTray:
		switch lParam {
		case wmLButtonUp, wmLButtonDbl:
			showFromTray()
		case wmRButtonUp:
			showTrayMenu(hwnd)
		}
		return 0
	case wmCommand:
		switch uint16(wParam) {
		case trayOpenID:
			showFromTray()
		case trayExitID:
			if trayHooks.Quit != nil {
				trayHooks.Quit()
			}
		}
		return 0
	case wmDestroy:
		prev := trayPrevProc
		DestroyTray()
		if prev != 0 {
			r, _, _ := procCallWindowProcW.Call(prev, hwnd, msg, wParam, lParam)
			return r
		}
		return 0
	}
	if trayPrevProc != 0 {
		r, _, _ := procCallWindowProcW.Call(trayPrevProc, hwnd, msg, wParam, lParam)
		return r
	}
	return 0
}

func showFromTray() {
	if trayHooks.Show != nil {
		trayHooks.Show()
	}
}

func showTrayMenu(hwnd uintptr) {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)
	procAppendMenuW.Call(menu, mfString, trayOpenID, uintptr(unsafe.Pointer(utf16Ptr("打开界面"))))
	procAppendMenuW.Call(menu, mfString, trayExitID, uintptr(unsafe.Pointer(utf16Ptr("退出"))))
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWnd.Call(hwnd)
	procTrackPopupMenu.Call(menu, tpmRightButton, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
}

func loadAppIcon() uintptr {
	var buf [260]uint16
	n, _, _ := procGetModuleFileName.Call(0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return 0
	}
	var large, small uintptr
	procExtractIconExW.Call(uintptr(unsafe.Pointer(&buf[0])), 0, uintptr(unsafe.Pointer(&large)), uintptr(unsafe.Pointer(&small)), 1)
	if small != 0 {
		if large != 0 && large != small {
			procDestroyIcon.Call(large)
		}
		return small
	}
	return large
}

func copyUTF16(dst []uint16, s string) {
	src, err := syscall.UTF16FromString(s)
	if err != nil {
		return
	}
	n := len(src)
	if n > len(dst) {
		n = len(dst)
		src[n-1] = 0
	}
	copy(dst, src[:n])
}
