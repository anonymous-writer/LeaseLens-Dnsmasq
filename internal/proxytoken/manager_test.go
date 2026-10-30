package proxytoken

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewInitializesMissingAndEmptyFile(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "empty"}[empty], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "token.json")
			if empty {
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := New(path, 30*24*time.Hour, 5*time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var state State
			if err := json.Unmarshal(b, &state); err != nil {
				t.Fatal(err)
			}
			if len(state.Current) != 100 {
				t.Fatalf("token length = %d, want 100", len(state.Current))
			}
			if state.CreatedAt.IsZero() {
				t.Fatal("created_at is zero")
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0640 {
				t.Fatalf("mode = %04o, want 0640", info.Mode().Perm())
			}
		})
	}
}

func TestNewRejectsNonEmptyInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(path, 30*24*time.Hour, 5*time.Minute); err == nil || !strings.Contains(err.Error(), "decode proxy token file") {
		t.Fatalf("New error = %v, want decode error", err)
	}
}

func TestRotationGracePeriod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	m, err := New(path, time.Hour, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	old := m.state.Current
	now := time.Now().Add(2 * time.Hour)
	if err := m.RotateIfDue(now); err != nil {
		t.Fatal(err)
	}
	if m.state.Current == old {
		t.Fatal("token did not rotate")
	}
	if !m.Valid(old, now.Add(4*time.Minute)) {
		t.Fatal("previous token should be accepted during grace")
	}
	if m.Valid(old, now.Add(6*time.Minute)) {
		t.Fatal("previous token should be rejected after grace")
	}
}
