package app

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUISettingsReadReturnsDatabaseErrors(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store := newUISettingsStore(db)
	if _, err := store.read(); err == nil {
		t.Fatal("read() should return an error when the database is closed")
	}
}

func TestUISettingsGetReturnsServerErrorWhenDatabaseUnavailable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ui-settings", nil)
	resp := httptest.NewRecorder()
	newUISettingsStore(db).handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%q", resp.Code, http.StatusInternalServerError, resp.Body.String())
	}
}
