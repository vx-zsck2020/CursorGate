package cursor

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestApplyModelsToTempDB(t *testing.T) {
	src := os.Getenv("APPDATA")
	if src == "" {
		t.Skip("no APPDATA")
	}
	src = filepath.Join(src, "Cursor", "User", "globalStorage", "state.vscdb")
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "state.vscdb")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := ApplyModelsTo(path, "http://127.0.0.1:8900/v1", []string{"deepseek-v4.1-flash", "hy3", "composer-2.5", "gpt-5.6", "x-ai/skip"}); err != nil {
		t.Fatal(err)
	}
	obj, err := readAppUser(path)
	if err != nil {
		t.Fatal(err)
	}
	if obj["openAIBaseUrl"] != "http://127.0.0.1:8900/v1" {
		t.Fatalf("base=%v", obj["openAIBaseUrl"])
	}
	if obj["useOpenAIKey"] != true {
		t.Fatal("useOpenAIKey")
	}
	ai := obj["aiSettings"].(map[string]any)
	enabled := asStringSlice(ai["modelOverrideEnabled"])
	if !contains(enabled, "hy3") || !contains(enabled, "composer-2.5") || !contains(enabled, "gpt-5.6") || contains(enabled, "x-ai/skip") {
		t.Fatalf("enabled=%v", enabled)
	}
	disabled := asStringSlice(ai["modelOverrideDisabled"])
	if contains(disabled, "composer-2.5") {
		t.Fatalf("composer should leave disabled: %v", disabled)
	}
	if !contains(disabled, "default") || !contains(disabled, "grok-4.7") {
		t.Fatalf("official names should stay disabled: %v", disabled)
	}
	catalog := asMapSlice(obj["availableDefaultModels2"])
	if len(catalog) == 0 {
		t.Fatal("empty catalog")
	}
	if catalog[0]["name"] != "deepseek-v4.1-flash" {
		t.Fatalf("custom models should lead catalog, first=%v", catalog[0]["name"])
	}
	names := map[string]map[string]any{}
	for _, m := range catalog {
		name, _ := m["name"].(string)
		names[name] = m
		if _, ok := m["vendor"]; ok {
			t.Fatalf("%s still has vendor", name)
		}
		if v, _ := m["isUserAdded"].(bool); !v {
			t.Fatalf("%s should be user-added", name)
		}
		idx, _ := m["namedModelSectionIndex"].(float64)
		if int(idx) != 0 {
			t.Fatalf("%s namedModelSectionIndex=%v", name, m["namedModelSectionIndex"])
		}
	}
	if names["grok-4.7"] != nil {
		t.Fatal("official leftover grok-4.7 must not stay in catalog")
	}
	if names["default"] != nil {
		t.Fatal("Auto/default must not stay in catalog")
	}
	if names["composer-2.5"] == nil {
		t.Fatal("composer-2.5 missing from custom catalog")
	}
	if names["gpt-5.6"] == nil {
		t.Fatal("gpt-5.6 missing from catalog")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM ItemTable WHERE key = ?", appUserKey).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("key count %d", n)
	}
	if !contains(asStringSlice(ai["userAddedModels"]), "gpt-5.6") {
		t.Fatalf("userAdded=%v", ai["userAddedModels"])
	}
}

func TestVersionReadsPackage(t *testing.T) {
	if v := Version(); v == "" {
		t.Fatal("empty version")
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
