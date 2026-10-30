package proxytoken

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type State struct {
	Current       string    `json:"current"`
	CreatedAt     time.Time `json:"created_at"`
	Previous      string    `json:"previous,omitempty"`
	PreviousUntil time.Time `json:"previous_until,omitempty"`
}
type Manager struct {
	mu              sync.Mutex
	path            string
	validity, grace time.Duration
	state           State
}

// New opens an existing token state or creates a fresh one. Corrupt existing
// files fail closed; they are never silently replaced.
func New(path string, validity, grace time.Duration) (*Manager, error) {
	if path == "" {
		return nil, errors.New("proxy token file path is required")
	}
	if validity <= 0 || grace < 0 || grace >= validity {
		return nil, errors.New("invalid proxy token validity/grace period")
	}
	m := &Manager{path: path, validity: validity, grace: grace}
	b, err := os.ReadFile(path)
	if err == nil && len(bytes.TrimSpace(b)) > 0 {
		if err = json.Unmarshal(b, &m.state); err != nil {
			return nil, fmt.Errorf("decode proxy token file: %w", err)
		}
		if len(m.state.Current) != 100 || m.state.CreatedAt.IsZero() {
			return nil, errors.New("proxy token file has invalid current token state")
		}
		return m, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	// A missing OR zero-byte/whitespace-only file is an uninitialized state.
	// Initialize it automatically. Non-empty malformed JSON above still fails closed.
	token, err := newToken()
	if err != nil {
		return nil, err
	}
	m.state = State{Current: token, CreatedAt: time.Now().UTC()}
	if err = m.writeLocked(); err != nil {
		return nil, err
	}
	return m, nil
}
func newToken() (string, error) {
	b := make([]byte, 50)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func (m *Manager) writeLocked() error {
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}
	// Create the temporary file in the destination directory so rename is atomic
	// and the directory's setgid bit gives it the shared www-data group.
	f, err := os.CreateTemp(dir, ".proxy-token-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0640); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp, m.path); err != nil {
		return err
	}
	if d, e := os.Open(dir); e == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

func (m *Manager) RotateIfDue(now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if now.Sub(m.state.CreatedAt) < m.validity {
		return nil
	}
	t, e := newToken()
	if e != nil {
		return e
	}
	m.state.Previous = m.state.Current
	m.state.PreviousUntil = now.Add(m.grace)
	m.state.Current = t
	m.state.CreatedAt = now.UTC()
	return m.writeLocked()
}
func (m *Manager) Run(stop <-chan struct{}) {
	interval := time.Minute
	if m.validity/4 < interval {
		interval = m.validity / 4
	}
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			if err := m.RotateIfDue(now); err != nil {
				fmt.Fprintf(os.Stderr, "proxy token rotation failed: %v\n", err)
			}
		}
	}
}
func (m *Manager) Valid(token string, now time.Time) bool {
	if len(token) != 100 {
		return false
	}
	b, e := os.ReadFile(m.path)
	if e != nil {
		return false
	}
	var s State
	if json.Unmarshal(b, &s) != nil {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.Current)) == 1 {
		return true
	}
	return s.Previous != "" && now.Before(s.PreviousUntil) && subtle.ConstantTimeCompare([]byte(token), []byte(s.Previous)) == 1
}
func (m *Manager) Path() string { return m.path }
