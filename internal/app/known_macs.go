package app

import (
	"database/sql"
	"net/http"
	"strings"
	"sync"
)

// KnownMACs persists remembered devices in SQLite.
type KnownMACs struct {
	mu sync.Mutex
	db *sql.DB
}

func newKnownMACs(db *sql.DB) *KnownMACs { return &KnownMACs{db: db} }
func (k *KnownMACs) has(mac string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	var found int
	err := k.db.QueryRow(`SELECT 1 FROM known_devices WHERE mac=? COLLATE NOCASE`, strings.ToUpper(strings.TrimSpace(mac))).Scan(&found)
	return err == nil
}
func (k *KnownMACs) mark(mac string) error {
	mac = strings.ToUpper(strings.TrimSpace(mac))
	if !macRE.MatchString(mac) {
		return http.ErrNotSupported
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	_, err := k.db.Exec(`INSERT OR IGNORE INTO known_devices(mac) VALUES(?)`, mac)
	return err
}
func (k *KnownMACs) markMany(macs []string) (int, error) {
	unique := make(map[string]struct{}, len(macs))
	for _, m := range macs {
		m = strings.ToUpper(strings.TrimSpace(m))
		if macRE.MatchString(m) {
			unique[m] = struct{}{}
		}
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	tx, err := k.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for m := range unique {
		if _, err = tx.Exec(`INSERT OR IGNORE INTO known_devices(mac) VALUES(?)`, m); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return len(unique), nil
}
func (k *KnownMACs) unmark(mac string) error {
	mac = strings.ToUpper(strings.TrimSpace(mac))
	if !macRE.MatchString(mac) {
		return http.ErrNotSupported
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	_, err := k.db.Exec(`DELETE FROM known_devices WHERE mac=? COLLATE NOCASE`, mac)
	return err
}
func (k *KnownMACs) reset() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	_, err := k.db.Exec(`DELETE FROM known_devices`)
	return err
}
