//go:build windows

package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func Apply(st Status) error {
	if st.AssetURL == "" {
		return fmt.Errorf("没有可下载的安装包")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)
	newPath := filepath.Join(dir, "CursorGate.new.exe")
	if err := DownloadTo(st.AssetURL, newPath, st.SHA256); err != nil {
		return err
	}
	bat := filepath.Join(os.TempDir(), "cursorgate-update.bat")
	body := fmt.Sprintf("@echo off\r\n"+
		"set SRC=%s\r\n"+
		"set DST=%s\r\n"+
		":wait\r\n"+
		"ping -n 2 127.0.0.1 >nul\r\n"+
		"copy /Y \"%%SRC%%\" \"%%DST%%\" >nul\r\n"+
		"if errorlevel 1 goto wait\r\n"+
		"del \"%%SRC%%\" >nul 2>nul\r\n"+
		"start \"\" \"%%DST%%\"\r\n"+
		"del \"%%~f0\"\r\n", newPath, exe)
	if err := os.WriteFile(bat, []byte(body), 0644); err != nil {
		return err
	}
	cmd := exec.Command("cmd.exe", "/C", "start", "", bat)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
