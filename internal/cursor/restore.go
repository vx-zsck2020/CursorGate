package cursor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RestoreResult struct {
	Message string `json:"message"`
	Status  Status `json:"status"`
}

func RestoreEnvironment() (RestoreResult, error) {
	wasRunning := Running()
	if wasRunning {
		if err := Quit(20 * time.Second); err != nil {
			return RestoreResult{}, fmt.Errorf("无法退出 Cursor：%w", err)
		}
	}
	var notes []string
	if path, err := latestVscdbBackup(); err == nil {
		dst, err := DefaultDBPath()
		if err != nil {
			return RestoreResult{}, err
		}
		if err := copyFile(path, dst); err != nil {
			return RestoreResult{}, fmt.Errorf("还原状态库失败：%w", err)
		}
		notes = append(notes, "已还原 Cursor 状态库")
	} else {
		notes = append(notes, "没有找到状态库备份，跳过 vscdb")
	}
	if err := RestoreCatalogRefresh(); err != nil {
		if wasRunning {
			_ = Launch()
		}
		return RestoreResult{}, err
	}
	notes = append(notes, "已撤回官方模型刷新短路")
	if wasRunning {
		if err := Launch(); err != nil {
			st, _ := ReadStatus()
			return RestoreResult{Message: strings.Join(notes, "；"), Status: st}, fmt.Errorf("已还原，但启动 Cursor 失败：%w", err)
		}
		deadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(deadline) {
			if Running() {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		notes = append(notes, "已重启 Cursor")
	}
	st, _ := ReadStatus()
	return RestoreResult{Message: strings.Join(notes, "；"), Status: st}, nil
}

func RestoreCatalogRefresh() error {
	dir := InstallDir()
	if dir == "" {
		return fmt.Errorf("找不到 Cursor 安装目录")
	}
	if Running() {
		return fmt.Errorf("Cursor 仍在运行，无法撤回 workbench 补丁")
	}
	for _, path := range workbenchFiles(dir) {
		if err := unpatchFile(path); err != nil {
			return err
		}
	}
	return nil
}

func unpatchFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读 %s: %w", filepath.Base(path), err)
	}
	src := string(raw)
	next, err := unpatchRefreshSource(src)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	if next == src {
		return nil
	}
	tmp := path + ".cg.tmp"
	if err := os.WriteFile(tmp, []byte(next), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func unpatchRefreshSource(src string) (string, error) {
	if !skipRefreshApplied(src) {
		return src, nil
	}
	n := strings.Count(src, refreshPatch)
	if n == 0 {
		return src, nil
	}
	if n != 1 {
		return "", fmt.Errorf("refreshDefaultModels 撤回点不唯一：%d", n)
	}
	return strings.Replace(src, refreshPatch, refreshNeedle, 1), nil
}

func latestVscdbBackup() (string, error) {
	dirs := []string{backupDir("")}
	if path, err := DefaultDBPath(); err == nil {
		dirs = append(dirs, filepath.Dir(path))
	}
	return latestBackupIn(dirs)
}

func latestBackupIn(dirs []string) (string, error) {
	var best string
	var bestTime time.Time
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasPrefix(name, "state.vscdb.bak_cursorgate_") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if best == "" || info.ModTime().After(bestTime) {
				best = filepath.Join(dir, name)
				bestTime = info.ModTime()
			}
		}
	}
	if best == "" {
		return "", os.ErrNotExist
	}
	return best, nil
}

func backupDir(fallback string) string {
	dir := filepath.Join(`D:\Program Files\cursor`, ".wb_backup")
	if err := os.MkdirAll(dir, 0755); err != nil {
		if fallback != "" {
			return fallback
		}
		return dir
	}
	return dir
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".cg.tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
