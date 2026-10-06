package cursor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPatchRefreshSource(t *testing.T) {
	src := "prefix;" + refreshNeedle + "tail"
	got, err := patchRefreshSource(src)
	if err != nil {
		t.Fatal(err)
	}
	if !skipRefreshApplied(got) {
		t.Fatal("patch marker missing")
	}
	if got == src {
		t.Fatal("source unchanged")
	}
	again, err := patchRefreshSource(got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatal("second apply should be idempotent")
	}
}

func TestPatchRefreshSourceMissing(t *testing.T) {
	if _, err := patchRefreshSource("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestInspectWorkbenchDir(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "resources", "app", "out", "vs", "workbench")
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	desktop := filepath.Join(base, "workbench.desktop.main.js")
	glass := filepath.Join(base, "workbench.glass.main.js")
	src := "prefix;" + refreshNeedle + "tail"
	if err := os.WriteFile(desktop, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(glass, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	st := inspectWorkbenchDir(dir)
	if !st.Patchable || st.Applied {
		t.Fatalf("unpatched should be patchable, got %+v", st)
	}
	patched, err := patchRefreshSource(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(desktop, []byte(patched), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(glass, []byte(patched), 0600); err != nil {
		t.Fatal(err)
	}
	st = inspectWorkbenchDir(dir)
	if !st.Patchable || !st.Applied {
		t.Fatalf("patched should be applied, got %+v", st)
	}
	st = inspectWorkbenchDir(t.TempDir())
	if st.Patchable {
		t.Fatalf("missing files should not be patchable, got %+v", st)
	}
	if err := os.WriteFile(desktop, []byte("nope"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(glass, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	st = inspectWorkbenchDir(dir)
	if st.Patchable {
		t.Fatalf("needle mismatch should not be patchable, got %+v", st)
	}
}

func TestLiveWorkbenchNeedle(t *testing.T) {
	dir := InstallDir()
	if dir == "" {
		t.Skip("no Cursor install")
	}
	for _, path := range workbenchFiles(dir) {
		raw, err := osReadFile(path)
		if err != nil {
			t.Fatal(path, err)
		}
		src := string(raw)
		if skipRefreshApplied(src) {
			continue
		}
		n := countNeedle(src)
		if n != 1 {
			t.Fatalf("%s refresh needle count=%d", path, n)
		}
	}
}

func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
