package app

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// UISettings stores browser-independent preferences in SQLite.
type UISettings struct {
	Language    string                     `json:"language"`
	Visible     map[string]bool            `json:"visible"`
	Preferences map[string]json.RawMessage `json:"preferences"`
}
type uiSettingsStore struct {
	db *sql.DB
	mu sync.Mutex
}

func newUISettingsStore(db *sql.DB) *uiSettingsStore { return &uiSettingsStore{db: db} }
func (s *uiSettingsStore) read() UISettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := UISettings{Language: "de", Visible: map[string]bool{}, Preferences: map[string]json.RawMessage{}}
	var lang, raw, prefs string
	err := s.db.QueryRow(adaptSQL(`SELECT language,visible_json,preferences_json FROM ui_settings WHERE id=1`)).Scan(&lang, &raw, &prefs)
	if err != nil {
		return out
	}
	if lang == "de" || lang == "en" || lang == "es" {
		out.Language = lang
	}
	if json.Unmarshal([]byte(raw), &out.Visible) != nil || out.Visible == nil {
		out.Visible = map[string]bool{}
	}
	if json.Unmarshal([]byte(prefs), &out.Preferences) != nil || out.Preferences == nil {
		out.Preferences = map[string]json.RawMessage{}
	}
	return out
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
			if settings.Preferences == nil {
				settings.Preferences = map[string]json.RawMessage{}
			}
			visible, err := json.Marshal(settings.Visible)
			if err != nil {
				http.Error(w, "invalid settings", http.StatusBadRequest)
				return
			}
			preferences, err := json.Marshal(settings.Preferences)
			if err != nil {
				http.Error(w, "invalid preferences", http.StatusBadRequest)
				return
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			_, err = s.db.Exec(adaptSQL(`INSERT INTO ui_settings(id,language,visible_json,preferences_json,updated_at) VALUES(1,?,?,?,?) ON CONFLICT(id) DO UPDATE SET language=excluded.language,visible_json=excluded.visible_json,preferences_json=excluded.preferences_json,updated_at=excluded.updated_at`), settings.Language, string(visible), string(preferences), time.Now().Format(time.RFC3339))
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
