# Native IPv4 DHCP lease release

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

The app only offers release for an IPv4 address/MAC pair that is present in the current lease file. It retrieves that record's client ID and invokes `dhcp_release` with the configured interface. That utility sends a DHCPRELEASE message to the local dnsmasq server, causing dnsmasq to release the IPv4 lease. The app does not edit the lease file directly or restart dnsmasq.

This is not a network block or a kick: a client that remains connected can immediately request another lease. The action does not support IPv6. Because the web UI has no authentication, keep it on localhost or a trusted network, especially when enabling this privileged action.

The supported utility is provided by the `dnsmasq-utils` package and must run as root. See the [`dhcp_release(1)` man page](https://manpages.debian.org/bookworm/dnsmasq-utils/dhcp_release.1.en.html).
