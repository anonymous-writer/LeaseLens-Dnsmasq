# Persistent working directory

LeaseLens supports a `working_dir` setting in its YAML-style configuration. When
set, the following defaults are resolved relative to that directory:

- `known-macs.json`
- `ui-settings.json`
- `dhcp-events.log`

Example for a native Raspberry Pi installation:

```yaml
working_dir: "/home/pi/leaselens-dnsmasq"
```

Explicit `known_macs_file`, `ui_settings_file`, and `dhcp_events_file` entries
remain supported and override the derived defaults. This preserves compatibility
with existing installations. The worker diagnostic log (`dnsmasq-ntfy-worker.log`)
and notification environment file (`dnsmasq-ntfy.env`) are intentionally not
managed by LeaseLens' working-directory setting.

This setting only centralizes file paths; it does not migrate the current JSON
and DHCP-event files into SQLite. A database migration must separately copy and
validate existing data before the old files can safely be retired.


## Hinweis zur nativen PostgreSQL-Konfiguration

`working_dir` ist für neue Installationen mit expliziten Datenbank- und Dateipfaden nicht erforderlich. Es bleibt aus Gründen der Abwärtskompatibilität im Programm erhalten. Für den DHCP-Aufruf gehört der Skriptpfad in die dnsmasq-Konfiguration, zum Beispiel:

```ini
dhcp-script=/usr/local/sbin/dnsmasq-ntfy-dhcp-ipv4.sh
```

`dhcp_script_path` ist daher keine LeaseLens-YAML-Einstellung: dnsmasq muss den Skriptpfad selbst kennen.
