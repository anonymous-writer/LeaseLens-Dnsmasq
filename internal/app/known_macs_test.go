package app

import (
	"database/sql"
	"testing"
)

func TestKnownMACsHasReturnsDatabaseError(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := newKnownMACs(db).has("AA:BB:CC:DD:EE:FF"); err == nil {
		t.Fatal("has() should report a database error instead of treating the device as unknown")
	}
}
