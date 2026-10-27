package app

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

var activeDatabaseType = "sqlite"

// adaptSQL translates the small set of SQLite query conventions used by LeaseLens
// to PostgreSQL placeholders and conflict clauses.
func adaptSQL(query string) string {
	if activeDatabaseType != "postgres" {
		return query
	}
	query = strings.ReplaceAll(query, " COLLATE NOCASE", "")
	query = strings.ReplaceAll(query, "INSERT OR IGNORE INTO", "INSERT INTO")
	query = strings.ReplaceAll(query, "INSERT OR REPLACE INTO", "INSERT INTO")
	var b strings.Builder
	idx := 0
	for _, r := range query {
		if r == '?' {
			idx++
			b.WriteString("$" + strconv.Itoa(idx))
		} else {
			b.WriteRune(r)
		}
	}
	query = b.String()
	if strings.Contains(query, "INSERT INTO known_devices(mac) VALUES($1)") {
		query += " ON CONFLICT(mac) DO NOTHING"
	}
	if strings.Contains(query, "INSERT INTO ui_settings(id,language,visible_json) VALUES(1,$1,$2)") {
		query += " ON CONFLICT(id) DO NOTHING"
	}
	if strings.Contains(query, "INSERT INTO dhcp_events(event_key,event_time,action,ip_address,mac,hostname) VALUES($1,$2,$3,$4,$5,$6)") {
		query += " ON CONFLICT(event_key) DO NOTHING"
	}
	if strings.Contains(query, "INSERT INTO app_metadata(key,value) VALUES($1,$2)") {
		query += " ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value"
	}
	return query
}

// openDatabase opens the single persistent LeaseLens database and creates its schema.
// Legacy JSON files are imported transactionally and are deliberately not deleted.
func openDatabase(path, knownMACsPath, uiSettingsPath, dhcpEventsPath string) (*sql.DB, error) {
	return openConfiguredDatabase("sqlite", path, "", "", "", "", "", "", knownMACsPath, uiSettingsPath, dhcpEventsPath)
}

func openConfiguredDatabase(dbType, path, pgHost, pgPort, pgName, pgUser, pgPassword, pgSSLMode, knownMACsPath, uiSettingsPath, dhcpEventsPath string) (*sql.DB, error) {
	dbType = strings.ToLower(strings.TrimSpace(dbType))
	if dbType == "" {
		dbType = "sqlite"
	}
	activeDatabaseType = dbType
	LogDebugf("database: initialization starting type=%q", dbType)
	var db *sql.DB
	var err error
	if dbType == "sqlite" {
		if strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("database path is empty")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
		db, err = sql.Open("sqlite", path)
	} else if dbType == "postgres" {
		if pgHost == "" || pgName == "" || pgUser == "" {
			return nil, fmt.Errorf("PostgreSQL requires postgres_host, postgres_database and postgres_user")
		}
		if pgPort == "" {
			pgPort = "5432"
		}
		if pgSSLMode == "" {
			pgSSLMode = "require"
		}
		password := pgPassword
		if password == "" {
			password = os.Getenv("POSTGRES_PASSWORD")
		}
		dsnURL := &url.URL{Scheme: "postgres", User: url.UserPassword(pgUser, password), Host: net.JoinHostPort(pgHost, pgPort), Path: "/" + pgName}
		query := dsnURL.Query()
		query.Set("sslmode", pgSSLMode)
		dsnURL.RawQuery = query.Encode()
		dsn := dsnURL.String()
		db, err = sql.Open("pgx", dsn)
	} else {
		return nil, fmt.Errorf("unsupported database_type %q (use sqlite or postgres)", dbType)
	}
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", dbType, err)
	}
	LogDebugf("database: driver opened type=%q; pinging server", dbType)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to %s database: %w", dbType, err)
	}
	LogDebugf("database: connectivity check successful type=%q", dbType)
	schema := []string{}
	if dbType == "sqlite" {
		for _, q := range []string{"PRAGMA journal_mode=DELETE", "PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=5000"} {
			if _, err = db.Exec(q); err != nil {
				db.Close()
				return nil, fmt.Errorf("configure SQLite: %w", err)
			}
		}
		schema = []string{
			`CREATE TABLE IF NOT EXISTS known_devices (mac TEXT PRIMARY KEY COLLATE NOCASE, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS ui_settings (id INTEGER PRIMARY KEY CHECK(id=1), language TEXT NOT NULL DEFAULT 'de', visible_json TEXT NOT NULL DEFAULT '{}', updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS dhcp_events (id INTEGER PRIMARY KEY AUTOINCREMENT, event_key TEXT NOT NULL UNIQUE, event_time TEXT NOT NULL, action TEXT NOT NULL, ip_address TEXT NOT NULL DEFAULT '', mac TEXT NOT NULL COLLATE NOCASE, hostname TEXT NOT NULL DEFAULT '', imported_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE INDEX IF NOT EXISTS idx_dhcp_events_mac_time ON dhcp_events(mac, event_time)`,
			`CREATE TABLE IF NOT EXISTS app_metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		}
	} else {
		schema = []string{
			`CREATE TABLE IF NOT EXISTS known_devices (mac TEXT PRIMARY KEY, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS ui_settings (id INTEGER PRIMARY KEY CHECK(id=1), language TEXT NOT NULL DEFAULT 'de', visible_json TEXT NOT NULL DEFAULT '{}', updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS dhcp_events (id BIGSERIAL PRIMARY KEY, event_key TEXT NOT NULL UNIQUE, event_time TIMESTAMPTZ NOT NULL, action TEXT NOT NULL, ip_address TEXT NOT NULL DEFAULT '', mac TEXT NOT NULL, hostname TEXT NOT NULL DEFAULT '', imported_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE INDEX IF NOT EXISTS idx_dhcp_events_mac_time ON dhcp_events(mac, event_time)`,
			`CREATE TABLE IF NOT EXISTS app_metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		}
	}
	for i, q := range schema {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			return nil, fmt.Errorf("create database schema statement %d: %w", i+1, err)
		}
	}
	LogDebugf("database: schema ready type=%q statements=%d", dbType, len(schema))
	// SQLite keeps the existing one-time legacy-file import for backward compatibility.
	// PostgreSQL starts with a fresh schema as explicitly requested; no data is migrated.
	if dbType == "sqlite" {
		if err := importLegacyKnownMACs(db, knownMACsPath); err != nil {
			db.Close()
			return nil, err
		}
		if err := importLegacyUISettings(db, uiSettingsPath); err != nil {
			db.Close()
			return nil, err
		}
	}
	// The legacy DHCP event log is imported into either backend. event_key is
	// a content hash, so repeated checks do not create duplicate rows.
	if err := importLegacyDHCPEvents(db, dhcpEventsPath); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// checkpointAndCloseDatabase closes the database pool during graceful shutdown.
// DELETE journal mode keeps the normal on-disk state in leaselens.db; SQLite may
// create a short-lived -journal file while a transaction is in progress. Never
// delete journal files manually while the application or another process runs.
func checkpointAndCloseDatabase(db *sql.DB) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		LogWarnf("close SQLite database: %v", err)
	} else {
		LogInfof("SQLite database closed")
	}
}

func alreadyImported(db *sql.DB, key string) (bool, error) {
	var v string
	err := db.QueryRow(adaptSQL(`SELECT value FROM app_metadata WHERE key=?`), key).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
func markImported(db *sql.DB, key string) error {
	_, err := db.Exec(adaptSQL(`INSERT OR REPLACE INTO app_metadata(key,value) VALUES(?,?)`), key, time.Now().Format(time.RFC3339))
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
		if _, err = tx.Exec(adaptSQL(`INSERT OR IGNORE INTO known_devices(mac) VALUES(?)`), mac); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(adaptSQL(`INSERT OR REPLACE INTO app_metadata(key,value) VALUES(?,?)`), key, time.Now().Format(time.RFC3339)); err != nil {
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
	if _, err = tx.Exec(adaptSQL(`INSERT OR IGNORE INTO ui_settings(id,language,visible_json) VALUES(1,?,?)`), s.Language, string(visible)); err != nil {
		return err
	}
	if _, err = tx.Exec(adaptSQL(`INSERT OR REPLACE INTO app_metadata(key,value) VALUES(?,?)`), key, time.Now().Format(time.RFC3339)); err != nil {
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
		if _, err = tx.Exec(adaptSQL(`INSERT OR IGNORE INTO dhcp_events(event_key,event_time,action,ip_address,mac,hostname) VALUES(?,?,?,?,?,?)`), key, ts.Format(time.RFC3339), strings.ToLower(fields[1]), ip, mac, hostname); err != nil {
			return err
		}
	}
	if err = sc.Err(); err != nil {
		return err
	}
	return tx.Commit()
}
