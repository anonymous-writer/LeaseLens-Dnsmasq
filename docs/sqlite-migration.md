# SQLite database migration

LeaseLens stores remembered MAC addresses and UI preferences in `leaselens.db`.
The database is created automatically at startup. When the legacy files exist,
`known-macs.json` and `ui-settings.json` are imported once and left in place as
backups; they are not deleted automatically.

DHCP events are imported into the `dhcp_events` table with a stable hash key, so
repeated imports do not create duplicate rows. The external dnsmasq script still
writes `dhcp-events.log`, which remains the live compatibility source for the
current lease table and is synced to SQLite when `/leases` is requested. Keep
that file until the dnsmasq event producer is explicitly migrated to write to
SQLite or to a LeaseLens event API.

Example native configuration:

```yaml
working_dir: "/home/pi/leaselens-dnsmasq"
# Optional; defaults to <working_dir>/leaselens.db
# database_path: "/home/pi/leaselens-dnsmasq/leaselens.db"
```

For Docker, mount the directory containing the database as a persistent volume.
Back up `leaselens.db` while LeaseLens is stopped, or use SQLite's backup API;
copying only the main file while WAL mode is active may omit recent changes.

Do not delete the old JSON files or DHCP event log until the new version has
started successfully and the imported records have been checked.
