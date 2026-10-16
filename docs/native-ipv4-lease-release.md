# Native DHCP lease removal (IPv4 and IPv6)

This optional action is only compiled into the standalone/native UI. The Docker build does not expose the endpoint or display the button.

## Configure

Set the interface on which dnsmasq serves the IPv4 lease in the native `config.yaml`:

```yaml
dhcp_interface: "eth1"
```

Alternatively set `DNSMASQ_INTERFACE=eth1` or pass `--dhcp-interface eth1`. The button stays hidden unless a valid interface name is configured.

Install the `dhcp_release` utility on Debian/Raspberry Pi OS:

```bash
sudo apt-get install dnsmasq-utils
```

Grant the account running the user service permission to execute only that utility without an interactive password prompt. For a service running as `pi`:

```bash
sudo visudo -f /etc/sudoers.d/dnsmasq-leases-ui
```

Add:

```sudoers
pi ALL=(root) NOPASSWD: /usr/bin/dhcp_release eth1 *
```

Validate the file and set its permissions:

```bash
sudo visudo -cf /etc/sudoers.d/dnsmasq-leases-ui
sudo chmod 0440 /etc/sudoers.d/dnsmasq-leases-ui
```

Change `pi` to the actual service account if needed. The application calls the command with `sudo -n`; it will not wait for a password prompt.

## Behavior and limits

The app only offers removal for an IPv4 address/MAC pair present in the current lease file. It retrieves that record's client ID and invokes `dhcp_release` with the configured interface. After a successful IPv4 release, the native build also removes IPv6 lease-file records with the same hostname, but only when that hostname belongs to exactly one IPv4 lease. Because dnsmasq's IPv6 records contain IAID/DUID rather than a MAC address, hostname matching is a conservative best-effort association. If the hostname is empty or ambiguous, IPv6 records are left untouched. The lease file is rewritten atomically; dnsmasq may subsequently recreate a lease for a client that reconnects.

This is not a network block or a kick: a client that remains connected can immediately request another lease. The IPv6 cleanup is a lease-file cleanup, not a DHCPv6 RELEASE exchange. Because the web UI has no authentication, keep it on localhost or a trusted network, especially when enabling this privileged action.

The supported utility is provided by the `dnsmasq-utils` package and must run as root. See the [`dhcp_release(1)` man page](https://manpages.debian.org/bookworm/dnsmasq-utils/dhcp_release.1.en.html).
