# PostgreSQL backend

LeaseLens supports `sqlite` (default) and `postgres` through `database_type` in
`config.yaml`. Both backends create the LeaseLens tables at startup.

Example:

```yaml
working_dir: "/home/pi/leaselens-dnsmasq"
database_type: "postgres"
postgres_host: "192.168.1.10"
postgres_port: "5432"
postgres_database: "leaselens"
postgres_user: "leaselens"
postgres_sslmode: "require"
```

Set the password in the environment, not in `config.yaml`:

```ini
Environment="POSTGRES_PASSWORD=replace-with-your-secret"
```

The PostgreSQL database itself must already exist, and the configured user must
have permission to create tables and indexes in it. `postgres_sslmode` accepts
PostgreSQL SSL modes such as `require`, `verify-ca`, `verify-full`, and `disable`
(use `disable` only on a trusted network where encryption is not required).

PostgreSQL starts with a fresh LeaseLens schema and does not import the SQLite
database or legacy JSON/log files. Switching `database_type` changes the backend;
it does not copy data between backends. Back up data before changing settings.

SQLite remains the default and keeps the existing one-time import of legacy
`known-macs.json`, `ui-settings.json`, and DHCP event logs.
