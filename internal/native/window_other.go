//go:build !windows

package native

type TrayHooks struct {
	Show func()
	Quit func()
}

func LockClientSize(title string, wantW, wantH int) {}

func StartTray(title string, hooks TrayHooks) {}

func DestroyTray() {}
