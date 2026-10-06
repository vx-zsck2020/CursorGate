package cursor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	refreshNeedle = "async refreshDefaultModels(){if(this.trySeedCatalogFromOwnKey(),!this._cursorAuthenticationService.isAuthenticated()){this._aiSettingsService.handleAvailableModelsChange();return}"
	refreshPatch  = "async refreshDefaultModels(){if(this._reactiveStorageService.applicationUserPersistentStorage.useOpenAIKey===!0){this._aiSettingsService.handleAvailableModelsChange();return}if(this.trySeedCatalogFromOwnKey(),!this._cursorAuthenticationService.isAuthenticated()){this._aiSettingsService.handleAvailableModelsChange();return}"
)

func workbenchFiles(installDir string) []string {
	base := filepath.Join(installDir, "resources", "app", "out", "vs", "workbench")
	return []string{
		filepath.Join(base, "workbench.desktop.main.js"),
		filepath.Join(base, "workbench.glass.main.js"),
	}
}

func skipRefreshApplied(src string) bool {
	return strings.Contains(src, "useOpenAIKey===!0){this._aiSettingsService.handleAvailableModelsChange();return}if(this.trySeedCatalogFromOwnKey()")
}

func patchRefreshSource(src string) (string, error) {
	if skipRefreshApplied(src) {
		return src, nil
	}
	n := strings.Count(src, refreshNeedle)
	if n == 0 {
		return "", fmt.Errorf("找不到 refreshDefaultModels 补丁点")
	}
	if n != 1 {
		return "", fmt.Errorf("refreshDefaultModels 补丁点不唯一：%d", n)
	}
	return strings.Replace(src, refreshNeedle, refreshPatch, 1), nil
}

func EnsureCatalogRefreshSkip() error {
	dir := InstallDir()
	if dir == "" {
		return fmt.Errorf("找不到 Cursor 安装目录")
	}
	if Running() {
		return fmt.Errorf("Cursor 仍在运行，无法写入 workbench 补丁")
	}
	for _, path := range workbenchFiles(dir) {
		if err := patchFile(path); err != nil {
			return err
		}
	}
	return nil
}

func patchFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读 %s: %w", filepath.Base(path), err)
	}
	src := string(raw)
	if skipRefreshApplied(src) {
		return nil
	}
	next, err := patchRefreshSource(src)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	if err := backupBytes(path, raw); err != nil {
		return err
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

func backupBytes(path string, raw []byte) error {
	bakDir := backupDir(filepath.Dir(path))
	name := filepath.Base(path) + ".bak_cursorgate_" + time.Now().Format("20060102_150405")
	return os.WriteFile(filepath.Join(bakDir, name), raw, 0600)
}

func countNeedle(src string) int {
	return strings.Count(src, refreshNeedle)
}
