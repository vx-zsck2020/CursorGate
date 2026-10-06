package cursor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUnpatchRefreshSource(t *testing.T) {
	src := "prefix;" + refreshNeedle + "tail"
	patched, err := patchRefreshSource(src)
	if err != nil {
		t.Fatal(err)
	}
	got, err := unpatchRefreshSource(patched)
	if err != nil {
		t.Fatal(err)
	}
	if got != src {
		t.Fatalf("unpatch mismatch\nwant %s\ngot  %s", src, got)
	}
	again, err := unpatchRefreshSource(src)
	if err != nil {
		t.Fatal(err)
	}
	if again != src {
		t.Fatal("unpatch clean source should be noop")
	}
}

func TestVersionCompatible(t *testing.T) {
	if !VersionCompatible(CompatibleVersion) {
		t.Fatal("self should match")
	}
	if VersionCompatible("3.23.22") || VersionCompatible("") {
		t.Fatal("mismatch should fail")
	}
}

func TestLatestVscdbBackupPicksNewest(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "state.vscdb.bak_cursorgate_20200101_000000")
	neu := filepath.Join(dir, "state.vscdb.bak_cursorgate_20990101_000000")
	if err := os.WriteFile(old, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(neu, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := latestBackupIn([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if got != neu {
		t.Fatalf("got %s want %s", got, neu)
	}
}
