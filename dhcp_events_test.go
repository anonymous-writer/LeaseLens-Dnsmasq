package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDHCPEventState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.log")
	content := "2026-10-20 18:40:59 +0200\told\t10.16.12.60\t14:35:b7:5c:8c:bf\tiphone-micha\n" +
		"2026-10-20 18:44:59 +0200\tadd\t10.16.12.60\t14:35:b7:5c:8c:bf\tiphone-micha\n" +
		"2026-10-20 18:45:59 +0200\tdel\t10.16.12.60\t14:35:b7:5c:8c:bf\tiphone-micha\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got := readDHCPEventState(path)["14:35:B7:5C:8C:BF"]
	if got.status != "del" {
		t.Fatalf("status = %q, want del", got.status)
	}
	if got.lastAdd.IsZero() {
		t.Fatal("last add timestamp should be populated")
	}
	if got.lastAdd.Format("2006-01-02 15:04:05") != "2026-10-20 18:44:59" {
		t.Fatalf("last add = %v", got.lastAdd)
	}
}
