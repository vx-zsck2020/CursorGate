package cursor

import (
	"os"
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
