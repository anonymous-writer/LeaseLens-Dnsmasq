package app

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// UISettings stores browser-independent preferences in the persistent data directory.
type UISettings struct {
	Language string          `json:"language"`
	Visible  map[string]bool `json:"visible"`
}

type uiSettingsStore struct {
	path string
	mu   sync.Mutex
}

func newUISettingsStore(path string) *uiSettingsStore { return &uiSettingsStore{path: path} }

func (s *uiSettingsStore) read() UISettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := UISettings{Language: "de", Visible: map[string]bool{}}
	b, err := os.ReadFile(s.path)
	if err != nil {
		return settings
	}
	if json.Unmarshal(b, &settings) != nil {
		return UISettings{Language: "de", Visible: map[string]bool{}}
	}
	if settings.Language != "de" && settings.Language != "en" && settings.Language != "es" {
		settings.Language = "de"
	}
	if settings.Visible == nil {
		settings.Visible = map[string]bool{}
	}
	return settings
}

func (s *uiSettingsStore) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(s.read())
		case http.MethodPut, http.MethodPost:
			var settings UISettings
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024)).Decode(&settings); err != nil {
				http.Error(w, "invalid settings", http.StatusBadRequest)
				return
			}
			if settings.Language != "de" && settings.Language != "en" && settings.Language != "es" {
				http.Error(w, "unsupported language", http.StatusBadRequest)
				return
			}
			if settings.Visible == nil {
				settings.Visible = map[string]bool{}
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
				http.Error(w, "cannot create settings directory", http.StatusInternalServerError)
				return
			}
			b, err := json.MarshalIndent(settings, "", "  ")
			if err == nil {
				err = os.WriteFile(s.path, append(b, '\n'), 0644)
			}
			if err != nil {
				http.Error(w, "cannot save settings", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.Header().Set("Allow", "GET, PUT, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
