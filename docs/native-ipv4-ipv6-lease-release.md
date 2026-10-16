# Native IPv4 + IPv6 lease release (Linux)

The application remains running as its normal service user. A narrowly scoped root helper performs the release operation because `/var/lib/dnsmasq/dnsmasq.leases` is root-managed on many systems.

## Install the helper

From the repository root:

```sh
sudo install -o root -g root -m 0755 scripts/dnsmasq-leases-ui-release /usr/local/sbin/dnsmasq-leases-ui-release
```

Check that dnsmasq-utils is installed and the executable exists:

```sh
command -v dhcp_release
ls -l /usr/bin/dhcp_release
```

The helper currently uses `/usr/bin/dhcp_release` and `/var/lib/dnsmasq/dnsmasq.leases`.

## Allow only this helper through sudo

Create `/etc/sudoers.d/dnsmasq-leases-ui` with `visudo`:

```sh
sudo visudo -f /etc/sudoers.d/dnsmasq-leases-ui
```

Add the following line, replacing `pi` if the service runs under another account:

```text
pi ALL=(root) NOPASSWD: /usr/local/sbin/dnsmasq-leases-ui-release
```

Then validate permissions:

```sh
sudo chmod 0440 /etc/sudoers.d/dnsmasq-leases-ui
sudo visudo -cf /etc/sudoers.d/dnsmasq-leases-ui
sudo -u pi sudo -n /usr/local/sbin/dnsmasq-leases-ui-release
```

The last command is expected to print usage and exit non-zero; it verifies that sudo permits the helper without prompting. Do not grant sudo access to a shell, Python interpreter, or arbitrary file editor.

## Restart and verify

Rebuild/reinstall the application using your usual deployment process, then restart the **user** service:

```sh
systemctl --user restart dnsmasq-leases-ui.service
journalctl --user -u dnsmasq-leases-ui.service -n 50 --no-pager
```

The helper validates the IPv4 lease/hostname association before calling `dhcp_release`. It recognizes the six-field dnsmasq IPv6 lease format (`expiry IAID IPv6 hostname DUID`), and prefers IPv6 entries whose DUID ends in the matched device MAC. If no such DUID association is available, it removes a hostname-matched IPv6 entry only when that hostname has exactly one IPv6 lease. Ambiguous entries are left untouched. IPv4 lease records have five fields. Review hostnames and test on one device before production use.
