package app

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// openDatabase opens the single persistent LeaseLens database and creates its schema.
// Legacy JSON files are imported transactionally and are deliberately not deleted.
func openDatabase(path, knownMACsPath, uiSettingsPath, dhcpEventsPath string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("database path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	pragmas := []string{"PRAGMA journal_mode=WAL", "PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=5000"}
	for _, q := range pragmas {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			return nil, fmt.Errorf("configure SQLite: %w", err)
		}
	}
	schema := []string{
		`CREATE TABLE IF NOT EXISTS known_devices (mac TEXT PRIMARY KEY COLLATE NOCASE, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS ui_settings (id INTEGER PRIMARY KEY CHECK(id=1), language TEXT NOT NULL DEFAULT 'de', visible_json TEXT NOT NULL DEFAULT '{}', updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS dhcp_events (id INTEGER PRIMARY KEY AUTOINCREMENT, event_key TEXT NOT NULL UNIQUE, event_time TEXT NOT NULL, action TEXT NOT NULL, ip_address TEXT NOT NULL DEFAULT '', mac TEXT NOT NULL COLLATE NOCASE, hostname TEXT NOT NULL DEFAULT '', imported_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE INDEX IF NOT EXISTS idx_dhcp_events_mac_time ON dhcp_events(mac, event_time)`,
		`CREATE TABLE IF NOT EXISTS app_metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
	}
	for _, q := range schema {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			return nil, fmt.Errorf("create database schema: %w", err)
		}
	}
	if err := importLegacyKnownMACs(db, knownMACsPath); err != nil {
		db.Close()
		return nil, err
	}
	if err := importLegacyUISettings(db, uiSettingsPath); err != nil {
		db.Close()
		return nil, err
	}
	if err := importLegacyDHCPEvents(db, dhcpEventsPath); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func alreadyImported(db *sql.DB, key string) (bool, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM app_metadata WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
func markImported(db *sql.DB, key string) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO app_metadata(key,value) VALUES(?,?)`, key, time.Now().Format(time.RFC3339))
	return err
}

func importLegacyKnownMACs(db *sql.DB, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	key := "migration_known_macs_v1:" + path
	done, err := alreadyImported(db, key)
	if err != nil || done {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read legacy known MACs: %w", err)
	}
	var macs []string
	if err := json.Unmarshal(data, &macs); err != nil {
		return fmt.Errorf("parse legacy known MACs %q: %w", path, err)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, mac := range macs {
		mac = strings.ToUpper(strings.TrimSpace(mac))
		if mac == "" {
			continue
		}
		if _, err = tx.Exec(`INSERT OR IGNORE INTO known_devices(mac) VALUES(?)`, mac); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`INSERT OR REPLACE INTO app_metadata(key,value) VALUES(?,?)`, key, time.Now().Format(time.RFC3339)); err != nil {
		return err
	}
	return tx.Commit()
}
func importLegacyUISettings(db *sql.DB, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	key := "migration_ui_settings_v1:" + path
	done, err := alreadyImported(db, key)
	if err != nil || done {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read legacy UI settings: %w", err)
	}
	var s UISettings
	if err = json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parse legacy UI settings %q: %w", path, err)
	}
	if s.Language != "de" && s.Language != "en" && s.Language != "es" {
		s.Language = "de"
	}
	if s.Visible == nil {
		s.Visible = map[string]bool{}
	}
	visible, err := json.Marshal(s.Visible)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT OR IGNORE INTO ui_settings(id,language,visible_json) VALUES(1,?,?)`, s.Language, string(visible)); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT OR REPLACE INTO app_metadata(key,value) VALUES(?,?)`, key, time.Now().Format(time.RFC3339)); err != nil {
		return err
	}
	return tx.Commit()
}

// DHCP event logs are still produced by the external dnsmasq script. Import them
// into SQLite for searchable history, but retain the source log for compatibility.
func importLegacyDHCPEvents(db *sql.DB, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open DHCP event log: %w", err)
	}
	defer f.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 4096), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}
		ts, e := time.Parse("2006-01-02 15:04:05 -0700", fields[0])
		if e != nil {
			continue
		}
		mac := strings.ToUpper(strings.TrimSpace(fields[3]))
		if mac == "" {
			continue
		}
		sum := sha256.Sum256([]byte(line))
		key := hex.EncodeToString(sum[:])
		hostname, ip := "", ""
		if len(fields) > 2 {
			ip = fields[2]
		}
		if len(fields) > 4 {
			hostname = fields[4]
		}
		if _, err = tx.Exec(`INSERT OR IGNORE INTO dhcp_events(event_key,event_time,action,ip_address,mac,hostname) VALUES(?,?,?,?,?,?)`, key, ts.Format(time.RFC3339), strings.ToLower(fields[1]), ip, mac, hostname); err != nil {
			return err
		}
	}
	if err = sc.Err(); err != nil {
		return err
	}
	return tx.Commit()
}
