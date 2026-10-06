package cursor

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func Running() bool {
	_, ok := firstCursorPID()
	return ok
}

func Version() string {
	dir := InstallDir()
	if dir == "" {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(dir, "resources", "app", "package.json"))
	if err != nil {
		return ""
	}
	var meta struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.Version)
}

func InstallDir() string {
	if pid, ok := firstCursorPID(); ok {
		if p := processPath(pid); p != "" {
			return filepath.Dir(p)
		}
	}
	for _, dir := range installCandidates() {
		if _, err := os.Stat(filepath.Join(dir, "Cursor.exe")); err == nil {
			return dir
		}
	}
	return ""
}

func Quit(timeout time.Duration) error {
	if !Running() {
		return nil
	}
	_ = exec.Command("taskkill", "/IM", "Cursor.exe", "/T").Run()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !Running() {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	_ = exec.Command("taskkill", "/F", "/IM", "Cursor.exe", "/T").Run()
	deadline = time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if !Running() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	if Running() {
		return os.ErrDeadlineExceeded
	}
	return nil
}

func Launch() error {
	dir := InstallDir()
	if dir == "" {
		return os.ErrNotExist
	}
	exe := filepath.Join(dir, "Cursor.exe")
	explorer := filepath.Join(os.Getenv("WINDIR"), "explorer.exe")
	if _, err := os.Stat(explorer); err == nil {
		cmd := exec.Command(explorer, exe)
		cmd.Dir = dir
		cmd.Env = filteredEnv()
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	cmd := exec.Command(exe)
	cmd.Dir = dir
	cmd.Env = filteredEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    false,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd.Start()
}

func Restart() error {
	if err := Quit(20 * time.Second); err != nil {
		return err
	}
	time.Sleep(800 * time.Millisecond)
	return Launch()
}

func installCandidates() []string {
	return []string{
		`D:\Program Files\cursor`,
		filepath.Join(os.Getenv("ProgramFiles"), "Cursor"),
		filepath.Join(os.Getenv("ProgramFiles"), "cursor"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "cursor"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Cursor"),
	}
}

func filteredEnv() []string {
	out := make([]string, 0, 32)
	for _, e := range os.Environ() {
		if strings.HasPrefix(strings.ToUpper(e), "ELECTRON_RUN_AS_NODE=") {
			continue
		}
		out = append(out, e)
	}
	return out
}

func firstCursorPID() (uint32, bool) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procCreate := kernel32.NewProc("CreateToolhelp32Snapshot")
	procFirst := kernel32.NewProc("Process32FirstW")
	procNext := kernel32.NewProc("Process32NextW")
	procClose := kernel32.NewProc("CloseHandle")

	const th32csSnapProcess = 0x00000002
	handle, _, _ := procCreate.Call(th32csSnapProcess, 0)
	if handle == 0 || handle == uintptr(syscall.InvalidHandle) {
		return 0, false
	}
	defer procClose.Call(handle)

	var e processEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	r, _, _ := procFirst.Call(handle, uintptr(unsafe.Pointer(&e)))
	self := uint32(os.Getpid())
	for r != 0 {
		name := syscall.UTF16ToString(e.ExeFile[:])
		if equalFold(name, "Cursor.exe") && e.ProcessID != self {
			return e.ProcessID, true
		}
		r, _, _ = procNext.Call(handle, uintptr(unsafe.Pointer(&e)))
	}
	return 0, false
}

func processPath(pid uint32) string {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)
	var buf [windows.MAX_PATH]uint16
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}

type processEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
