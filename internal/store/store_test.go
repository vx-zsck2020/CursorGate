package store

import (
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
