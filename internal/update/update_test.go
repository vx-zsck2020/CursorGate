package update

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewer(t *testing.T) {
	if !Newer("1.0.1", "1.0.0") {
		t.Fatal("1.0.1 should be newer")
	}
	if Newer("1.0.0", "1.0.0") {
		t.Fatal("equal is not newer")
	}
	if Newer("v1.0.0", "1.0.0") {
		t.Fatal("v prefix should normalize")
	}
	if !Newer("1.10.0", "1.9.9") {
		t.Fatal("1.10.0 > 1.9.9")
	}
	if Newer("1.0.0", "1.0.1") {
		t.Fatal("older should not win")
	}
}

func TestPickAsset(t *testing.T) {
	assets := []ghAsset{
		{Name: "notes.txt", BrowserDownloadURL: "http://x/notes"},
		{Name: "other.exe", BrowserDownloadURL: "http://x/other.exe"},
		{Name: "CursorGate.exe", BrowserDownloadURL: "http://x/CursorGate.exe"},
	}
	got := pickAsset(assets)
	if got == nil || got.Name != "CursorGate.exe" {
		t.Fatalf("got %#v", got)
	}
}

func TestDownloadToVerifiesSHA(t *testing.T) {
	payload := []byte("hello-gate")
	sum := sha256.Sum256(payload)
	want := hex.EncodeToString(sum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()
	dir := t.TempDir()
	dst := filepath.Join(dir, "CursorGate.exe")
	if err := DownloadTo(srv.URL, dst, want); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("content=%q", got)
	}
	if err := DownloadTo(srv.URL, dst, "deadbeef"); err == nil {
		t.Fatal("bad sha should fail")
	}
}

func TestCheckFromManifest(t *testing.T) {
	old := getFn
	getFn = func(url string) ([]byte, int, error) {
		if contains(url, "/releases/latest") {
			return []byte("missing"), 404, nil
		}
		if contains(url, "latest.json") {
			return []byte(`{"version":"9.9.9","notes":"n","downloadUrl":"http://x/CursorGate.exe","url":"http://rel"}`), 200, nil
		}
		return nil, 500, nil
	}
	defer func() { getFn = old }()

	st, err := Check()
	if err != nil {
		t.Fatal(err)
	}
	if st.Latest != "9.9.9" || !st.Available || st.AssetURL == "" {
		t.Fatalf("%+v", st)
	}
}

func TestCheckFromRelease(t *testing.T) {
	old := getFn
	getFn = func(url string) ([]byte, int, error) {
		if contains(url, "/releases/latest") {
			return []byte(`{"tag_name":"v1.2.0","body":"fix","html_url":"http://rel","assets":[{"name":"CursorGate.exe","browser_download_url":"http://x/CursorGate.exe"}]}`), 200, nil
		}
		return nil, 500, nil
	}
	defer func() { getFn = old }()
	st, err := Check()
	if err != nil {
		t.Fatal(err)
	}
	if st.Latest != "1.2.0" || !st.Available {
		t.Fatalf("%+v", st)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
