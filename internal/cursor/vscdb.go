package cursor

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const appUserKey = "src.vs.platform.reactivestorage.browser.reactiveStorageServiceImpl.persistentStorage.applicationUser"

var knownOfficial = []string{
	"default",
	"grok-4.7",
	"grok-4.6",
	"grok-4.5",
	"composer-2.5",
	"kimi-k3",
	"kimi-k2.7-code",
	"glm-5.2",
	"glm-5p3",
	"glm-5p3-flash",
}

const (
	CompatibleVersion = "3.23.23"
	DownloadURL       = "https://cursor.com/download"
)

type Status struct {
	Path              string   `json:"path"`
	Version           string   `json:"version"`
	Compatible        bool     `json:"compatible"`
	CompatibleVersion string   `json:"compatibleVersion"`
	DownloadURL       string   `json:"downloadUrl"`
	Patchable         bool     `json:"patchable"`
	PatchApplied      bool     `json:"patchApplied"`
	CompatNote        string   `json:"compatNote,omitempty"`
	UseOpenAIKey      bool     `json:"useOpenAIKey"`
	OpenAIBaseURL     string   `json:"openAIBaseUrl"`
	Enabled           []string `json:"enabled"`
	Disabled          []string `json:"disabled"`
	UserAdded         []string `json:"userAdded"`
	CursorRunning     bool     `json:"cursorRunning"`
}

func DefaultDBPath() (string, error) {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return "", fmt.Errorf("APPDATA 为空")
	}
	return filepath.Join(appdata, "Cursor", "User", "globalStorage", "state.vscdb"), nil
}

func ReadStatus() (Status, error) {
	path, err := DefaultDBPath()
	if err != nil {
		return Status{}, err
	}
	ver := Version()
	wb := InspectWorkbench()
	st := Status{
		Path:              path,
		CursorRunning:     Running(),
		Version:           ver,
		Compatible:        wb.Patchable,
		CompatibleVersion: CompatibleVersion,
		DownloadURL:       DownloadURL,
		Patchable:         wb.Patchable,
		PatchApplied:      wb.Applied,
		CompatNote:        wb.Reason,
	}
	obj, err := readAppUser(path)
	if err != nil {
		return st, err
	}
	st.UseOpenAIKey, _ = obj["useOpenAIKey"].(bool)
	st.OpenAIBaseURL, _ = obj["openAIBaseUrl"].(string)
	if ai, ok := obj["aiSettings"].(map[string]any); ok {
		st.Enabled = asStringSlice(ai["modelOverrideEnabled"])
		st.Disabled = asStringSlice(ai["modelOverrideDisabled"])
		st.UserAdded = asStringSlice(ai["userAddedModels"])
	}
	return st, nil
}

func ApplyModels(baseURL string, customIDs []string) error {
	path, err := DefaultDBPath()
	if err != nil {
		return err
	}
	if Running() {
		return fmt.Errorf("Cursor 仍在运行，无法写入状态库")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("找不到 Cursor 状态库: %w", err)
	}
	if err := backup(path); err != nil {
		return err
	}
	return ApplyModelsTo(path, baseURL, customIDs)
}

func ApplyModelsTo(path, baseURL string, customIDs []string) error {
	obj, err := readAppUser(path)
	if err != nil {
		return err
	}
	obj["useOpenAIKey"] = true
	obj["openAIBaseUrl"] = strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(obj["openAIBaseUrl"].(string), "/v1") {
		obj["openAIBaseUrl"] = obj["openAIBaseUrl"].(string) + "/v1"
	}

	catalog := asMapSlice(obj["availableDefaultModels2"])
	official := union(knownOfficial, officialNames(catalog))
	customSet := map[string]struct{}{}
	cleaned := make([]string, 0, len(customIDs))
	for _, id := range customIDs {
		id = strings.TrimSpace(id)
		if id == "" || strings.Contains(id, "/") || id == "default" {
			continue
		}
		if _, ok := customSet[id]; ok {
			continue
		}
		customSet[id] = struct{}{}
		cleaned = append(cleaned, id)
	}

	customEntries := make([]map[string]any, 0, len(cleaned))
	for _, id := range cleaned {
		customEntries = append(customEntries, userAddedEntry(id))
	}
	obj["availableDefaultModels2"] = customEntries

	ai, _ := obj["aiSettings"].(map[string]any)
	if ai == nil {
		ai = map[string]any{}
	}
	ai["userAddedModels"] = cleaned
	ai["modelOverrideEnabled"] = cleaned

	disabled := make([]string, 0, len(official)+1)
	disabled = appendUnique(disabled, "default")
	for _, name := range official {
		if _, ok := customSet[name]; ok {
			continue
		}
		disabled = appendUnique(disabled, name)
	}
	ai["modelOverrideDisabled"] = disabled
	obj["aiSettings"] = ai

	raw, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return writeAppUser(path, string(raw))
}

func userAddedEntry(id string) map[string]any {
	return map[string]any{
		"name":                               id,
		"defaultOn":                          false,
		"supportsAgent":                      true,
		"degradationStatus":                  0,
		"supportsThinking":                   true,
		"supportsImages":                     true,
		"supportsMaxMode":                    true,
		"supportsNonMaxMode":                 true,
		"clientDisplayName":                  id,
		"serverModelName":                    id,
		"isRecommendedForBackgroundComposer": false,
		"supportsPlanMode":                   true,
		"supportsSandboxing":                 true,
		"isUserAdded":                        true,
		"inputboxShortModelName":             id,
		"parameterDefinitions":               []any{},
		"variants":                           []any{},
		"legacySlugs":                        []any{},
		"idAliases":                          []any{},
		"namedModelSectionIndex":             0,
		"visibleInRoutedModelView":           true,
		"cloudAgentEffortModes":              []any{},
		"modelPickerBadges":                  []any{},
	}
}

func officialNames(catalog []map[string]any) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, m := range catalog {
		name, _ := m["name"].(string)
		if name == "" {
			continue
		}
		if v, ok := m["isUserAdded"].(bool); ok && v {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func union(a, b []string) []string {
	out := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		out = appendUnique(out, s)
	}
	for _, s := range b {
		out = appendUnique(out, s)
	}
	return out
}

func appendUnique(ss []string, v string) []string {
	for _, s := range ss {
		if s == v {
			return ss
		}
	}
	return append(ss, v)
}

func VersionCompatible(v string) bool {
	v = strings.TrimSpace(v)
	return v != "" && v == CompatibleVersion
}

func backup(path string) error {
	bakDir := backupDir(filepath.Dir(path))
	dst := filepath.Join(bakDir, "state.vscdb.bak_cursorgate_"+time.Now().Format("20060102_150405"))
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, src, 0600)
}

func readAppUser(path string) (map[string]any, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var value string
	err = db.QueryRow("SELECT value FROM ItemTable WHERE key = ?", appUserKey).Scan(&value)
	if err != nil {
		return nil, fmt.Errorf("读 applicationUser: %w", err)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(value), &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func writeAppUser(path, value string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("UPDATE ItemTable SET value = ? WHERE key = ?", value, appUserKey)
	return err
}

func asStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		if ss, ok := v.([]string); ok {
			return ss
		}
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func asMapSlice(v any) []map[string]any {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, x := range arr {
		if m, ok := x.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}
