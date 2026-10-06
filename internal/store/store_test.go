package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpsertEncryptsKey(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.Upsert(Provider{Name: "A", BaseURL: "http://127.0.0.1:1/v1", Enabled: true}, "sk-secret")
	if err != nil {
		t.Fatal(err)
	}
	if p.KeyEnc == "" || p.KeyEnc == "sk-secret" {
		t.Fatal("key should be encrypted")
	}
	got, err := st.DecryptKey(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-secret" {
		t.Fatalf("got %q", got)
	}
	p2, err := st.Upsert(Provider{ID: p.ID, Name: "A2", BaseURL: "http://127.0.0.1:1/v1", Enabled: true}, "")
	if err != nil {
		t.Fatal(err)
	}
	got, err = st.DecryptKey(p2)
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-secret" {
		t.Fatalf("empty key overwrite lost secret: %q", got)
	}
}

func TestPrefsAndPortPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Prefs().Theme != "dark" {
		t.Fatalf("default theme=%q", st.Prefs().Theme)
	}
	if err := st.SetTheme("light"); err != nil {
		t.Fatal(err)
	}
	if err := st.SetCloseBehavior("tray", true); err != nil {
		t.Fatal(err)
	}
	if err := st.SetPort(8912); err != nil {
		t.Fatal(err)
	}
	st2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	p := st2.Prefs()
	if p.Theme != "light" || p.CloseAction != "tray" || !p.RememberClose {
		t.Fatalf("prefs=%+v", p)
	}
	if st2.Port() != 8912 {
		t.Fatalf("port=%d", st2.Port())
	}
}

func TestCatchAllIsExclusive(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := st.Upsert(Provider{Name: "A", BaseURL: "http://127.0.0.1:1/v1", Enabled: true, CatchAll: true}, "sk-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := st.Upsert(Provider{Name: "B", BaseURL: "http://127.0.0.1:2/v1", Enabled: true, CatchAll: true}, "sk-b")
	if err != nil {
		t.Fatal(err)
	}
	gotFirst, ok := st.Get(first.ID)
	if !ok {
		t.Fatal("missing first")
	}
	if gotFirst.CatchAll {
		t.Fatal("old catch-all should be cleared")
	}
	if !second.CatchAll {
		t.Fatal("new catch-all should win")
	}
}

func TestOpenKeepsFirstCatchAll(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := []byte(`{"gatewayPort":8900,"providers":[{"id":"a","name":"A","baseUrl":"http://a/v1","catchAll":true},{"id":"b","name":"B","baseUrl":"http://b/v1","catchAll":true}]}`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	list := st.List()
	if len(list) != 2 {
		t.Fatalf("len=%d", len(list))
	}
	if !list[0].CatchAll || list[1].CatchAll {
		t.Fatalf("want only first catch-all, got %+v %+v", list[0], list[1])
	}
}
