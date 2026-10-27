/*
 * leaselens-dnsmasq frontend behavior.
 * All user-visible UI text is provided by static/translations.js.
 */
'use strict';

const THEME_KEY = 'dnsmasq-leases-theme';
const SORT_KEY = 'dnsmasq-leases-sort';
const LEASES_API_URL = '/leases';
const KNOWN_MACS_API_URL = '/known-macs';
const LANGUAGE_API = () => window.DnsmasqI18n;
const t = (key, variables) => LANGUAGE_API().t(key, variables);
const COUNTDOWN_RENDER_INTERVAL_MS = 1000;
const DEFAULT_SORT = { column: 'ipAddress', dir: 'asc' };

let leases = [];
let filterQuery = '';
let ipFilter = 'all';
let reservationFilter = 'all';
let statusFilter = 'all'; let groupFilter = 'all';
let sort = loadSort();
let refreshing = false;
let lastLoadError = null;
let lastSuccessfulRefresh = null;
let activeDetailLease = null;
const status = document.querySelector('#status');
const lastUpdated = document.querySelector('#last-updated');
const refreshState = document.querySelector('#refresh-state');

// Zeigt die lokale Systemzeit des Browsers im LeaseLens-Kopf an.
function updateBrandClock() {
  const now = new Date();
  const lang = { en: 'en-GB', de: 'de-DE', es: 'es-ES' }[LANGUAGE_API().language] || 'en-GB';
  const clock = document.querySelector('#system-clock');
  const date = document.querySelector('#system-date');
  if (clock) clock.textContent = new Intl.DateTimeFormat(lang, { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(now);
  if (date) date.textContent = new Intl.DateTimeFormat(lang, { dateStyle: 'medium' }).format(now);
}
updateBrandClock();
window.setInterval(updateBrandClock, 1000);

(() => {
  try {
    const theme = localStorage.getItem(THEME_KEY);
    if (theme === 'light' || theme === 'dark') document.documentElement.dataset.theme = theme;
  } catch (_) { /* Theme persistence is optional. */ }
})();

document.querySelector('#theme-toggle').addEventListener('click', () => {
  const current = document.documentElement.dataset.theme ??
    (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
  const next = current === 'dark' ? 'light' : 'dark';
  document.documentElement.dataset.theme = next;
  try { localStorage.setItem(THEME_KEY, next); } catch (_) { /* Theme persistence is optional. */ }
});

function loadSort() {
  try {
    const saved = JSON.parse(localStorage.getItem(SORT_KEY));
    if (saved?.column && saved?.dir) return saved;
  } catch (_) { /* Use the default sort when storage is unavailable or invalid. */ }
  return { ...DEFAULT_SORT };
}

function saveSort() {
  try { localStorage.setItem(SORT_KEY, JSON.stringify(sort)); } catch (_) { /* Optional persistence. */ }
}

function formatLease(lease) {
  if (lease.leaseExpiry === 0) return t('never');
  if (!Number.isFinite(Number(lease.leaseExpiry))) return lease.leasetime ?? '';
  const seconds = Number(lease.leaseExpiry) - Math.floor(Date.now() / 1000);
  if (seconds <= 0) return t('expired');
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return t('inDaysHours', { days, hours });
  if (hours > 0) return t('inHoursMinutes', { hours, minutes });
  if (minutes > 0) return t('inMinutes', { minutes });
  return t('inSeconds', { seconds });
}

function match(lease, query) {
  if (!query) return true;
  query = query.toLowerCase();
  const reservationTerms = lease.staticIP ? ['yes', 'ja'] : ['no', 'nein'];
  return [lease.ipAddress, lease.macAddress, lease.vendor, lease.privateMAC ? t('privateMACAddress') : '', lease.iaid, lease.duid, lease.name, lease.group, lease.leasetime, formatLease(lease),
    ...reservationTerms, lease.status].some(value => String(value ?? '').toLowerCase().includes(query));
}

function elapsedSeconds(value) {
  // The API's age string is formatted as e.g. "2d 3h 5m".
  // Convert it to a duration so sorting follows time, not the event label ("old"/"add"/"del").
  const text = String(value ?? '');
  const units = [
    [/([0-9]+)\s*d/i, 86400],
    [/([0-9]+)\s*h/i, 3600],
    [/([0-9]+)\s*m/i, 60],
    [/([0-9]+)\s*s/i, 1]
  ];
  let total = 0;
  let found = false;
  for (const [pattern, multiplier] of units) {
    const match = text.match(pattern);
    if (match) {
      total += Number(match[1]) * multiplier;
      found = true;
    }
  }
  return found ? total : null;
}

function sorted(items) {
  return [...items].sort((a, b) => {
    if (sort.column === 'lastDhcpEventAge') {
      const leftAge = elapsedSeconds(a.lastDhcpEventAge);
      const rightAge = elapsedSeconds(b.lastDhcpEventAge);
      // Entries without an event timestamp always go to the end.
      if (leftAge === null && rightAge !== null) return 1;
      if (rightAge === null && leftAge !== null) return -1;
      if (leftAge !== null && rightAge !== null && leftAge !== rightAge) {
        const result = leftAge - rightAge;
        return sort.dir === 'asc' ? result : -result;
      }
      // Equal ages are ordered by event type, with "old" after add/del.
      const rank = value => ({ add: 0, del: 1, old: 2 })[String(value ?? '').toLowerCase()] ?? 3;
      const result = rank(a.dhcpEventStatus) - rank(b.dhcpEventStatus);
      return sort.dir === 'asc' ? result : -result;
    }

    let left = a[sort.column];
    let right = b[sort.column];
    if (sort.column === 'staticIP') {
      left = left ? 1 : 0;
      right = right ? 1 : 0;
    }
    const result = String(left ?? '').localeCompare(String(right ?? ''), undefined,
      { numeric: true, sensitivity: 'base' });
    return sort.dir === 'asc' ? result : -result;
  });
}

function link(text, url) {
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.target = '_blank';
  anchor.rel = 'noopener noreferrer';
  anchor.textContent = text;
  return anchor;
}

function statusDescription(statusValue) {
  switch (statusValue) {
    case 'online': return t('httpHttpsReachable');
    case 'reachable': return t('pingReachableNoHTTP');
    case 'offline': return t('noResponse');
    default: return t('statusUnknown');
  }
}

function displayVendor(lease) {
  if (!validMac(lease.macAddress)) return '';
  if (lease.privateMAC) return t('privateMACAddress');
  return lease.vendor || t('vendorUnknown');
}

function statusCell(lease) {
  const cell = document.createElement('td');
  cell.className = 'device-status';
  cell.dataset.column = 'status';
  cell.title = statusDescription(lease.status);
  const dot = document.createElement('span');
  dot.className = `dot ${lease.status || 'unknown'}`;
  dot.title = statusDescription(lease.status);
  dot.setAttribute('role', 'img');
  dot.setAttribute('aria-label', dot.title);
  cell.append(dot);
  const label = document.createElement('span');
  label.className = 'status-label';
  label.textContent = ({ online: t('statusShortOnline'), reachable: t('statusShortPing'), offline: t('statusShortOffline'), unknown: t('unknown') })[lease.status] || t('unknown');
  cell.append(label);
  return cell;
}

function matchesFilters(lease) {
  const isIPv6 = String(lease.ipAddress ?? '').includes(':');
  if (ipFilter === 'ipv4' && isIPv6) return false;
  if (ipFilter === 'ipv6' && !isIPv6) return false;
  if (reservationFilter === 'reserved' && !lease.staticIP) return false;
  if (reservationFilter === 'dynamic' && lease.staticIP) return false;
  if (statusFilter !== 'all' && String(lease.status ?? 'unknown') !== statusFilter) return false;
  if (groupFilter !== 'all' && String(lease.group ?? '') !== groupFilter) return false;
  return true;
}

function updateLeaseSummary(shown, total) {
  status.classList.remove('error');
  if (shown !== total) {
    status.textContent = t('filteredSummary', { shown, total });
  } else {
    const language = LANGUAGE_API().language;
    status.textContent = ({
      de: 'Zeile anklicken, um Gerätedetails zu öffnen',
      en: 'Click a row to open device details',
      es: 'Haz clic en una fila para ver los detalles del dispositivo'
    })[language] || 'Click a row to open device details';
  }
}

function formattedTimestamp(date) {
  const locale = { en: 'en-GB', de: 'de-DE', es: 'es-ES' }[LANGUAGE_API().language] || 'en-GB';
  return new Intl.DateTimeFormat(locale, { dateStyle: 'short', timeStyle: 'medium' }).format(date);
}

function updateRefreshMeta() {
  lastUpdated.textContent = lastSuccessfulRefresh
    ? t('lastUpdated', { time: formattedTimestamp(lastSuccessfulRefresh) })
    : '';
  refreshState.classList.remove('error');
  if (refreshing) {
    refreshState.textContent = t('loadingLeases');
  } else if (lastLoadError) {
    refreshState.textContent = t('refreshFailedKeepingData', { error: lastLoadError });
    refreshState.classList.add('error');
  } else {
    refreshState.textContent = '';
  }
}

function updateStatusCards() {
  const count = key => leases.filter(lease => (lease.status || 'unknown') === key).length;
  document.querySelector('#summary-total').textContent = String(leases.length);
  document.querySelector('#summary-online').textContent = String(count('online'));
  document.querySelector('#summary-ping').textContent = String(count('reachable'));
  const now = Math.floor(Date.now() / 1000);
  const expiring = leases.filter(lease => {
    const expiry = Number(lease.leaseExpiry);
    return Number.isFinite(expiry) && expiry > now && expiry <= now + 86400;
  }).length;
  document.querySelector('#summary-expiring').textContent = String(expiring);
  document.querySelector('#summary-offline').textContent = String(count('offline'));
}

function render() {
  updateStatusCards();
  const groupSelect = document.querySelector('#group-filter');
  const selectedGroup = groupFilter;
  const groups = [...new Set(leases.map(lease => String(lease.group || '').trim()).filter(Boolean))].sort((a, b) => a.localeCompare(b));
  const hasGroups = groups.length > 0;
  groupSelect.hidden = !hasGroups;
  groupSelect.replaceChildren(new Option(t('allGroups'), 'all'));
  groups.forEach(group => groupSelect.add(new Option(group, group)));
  groupFilter = groups.includes(selectedGroup) ? selectedGroup : 'all';
  groupSelect.value = groupFilter;
  document.body.dataset.hasGroups = String(hasGroups);
  // Keep header and body columns in sync, including when groups are hidden.
  const groupColumnEnabled = document.body.dataset.showGroupColumn !== 'false';
  const groupHeader = document.querySelector('#group-header');
  groupHeader.hidden = !hasGroups;
  groupHeader.style.display = groupColumnEnabled && hasGroups ? '' : 'none';
  document.querySelectorAll('#leases tbody [data-column="group"]').forEach(cell => {
    cell.style.display = groupColumnEnabled && hasGroups ? '' : 'none';
  });
  const rows = sorted(leases.filter(lease => match(lease, filterQuery) && matchesFilters(lease)));
  const tbody = document.querySelector('#leases tbody');
  tbody.replaceChildren(...rows.map(lease => {
    const row = document.createElement('tr');
    row.className = `device-row${lease.isNew ? ' new-device' : ''}`;
    row.tabIndex = 0;
    row.setAttribute('aria-label', t('detailsFor', { device: lease.name || lease.ipAddress }));
    const hiddenColumns = new Set(JSON.parse(document.body.dataset.hiddenColumns || '[]'));
    const statusCellElement = statusCell(lease);
    if (hiddenColumns.has('status')) statusCellElement.style.display = 'none';
    row.appendChild(statusCellElement);
    const displayMAC = validMac(lease.macAddress) ? lease.macAddress : '—';
    const values = [lease.ipAddress, displayMAC, displayVendor(lease) || '—', lease.name, lease.group || '—',
      lease.staticIP ? t('yes') : t('no'), formatLease(lease)];
    const columnNames = ['ip', 'mac', 'vendor', 'name', 'group', 'reservation', 'lease-expiry'];
    if (document.body.dataset.dhcpEventsEnabled === 'true') {
      values.push(lease.dhcpEventStatus
        ? `${t(({ add: 'dhcpEventAdd', old: 'dhcpEventOld', del: 'dhcpEventDel' })[String(lease.dhcpEventStatus).toLowerCase()] || 'dhcpEventOld')}${lease.lastDhcpEventAge ? ` · ${t('lastDHCPEventAge', { age: lease.lastDhcpEventAge })}` : ''}`
        : '—');
      columnNames.push('dhcp-event');
    }
    values.forEach((value, index) => {
        const cell = document.createElement('td');
        cell.dataset.column = columnNames[index];
        if (hiddenColumns.has(columnNames[index]) || (columnNames[index] === 'group' && (!groupColumnEnabled || !hasGroups))) cell.style.display = 'none';
        if (index === 1 && lease.isNew && validMac(lease.macAddress)) {
          const badge = document.createElement('span');
          badge.className = 'new-badge';
          badge.textContent = t('newBadge');
          cell.append(badge, document.createTextNode(' '));
        }
        if (index === 0 && lease.webUrl) {
          cell.appendChild(link(lease.ipAddress, lease.webUrl));
        } else if (index === 3 && lease.webHostUrl) {
          cell.appendChild(link(lease.name, lease.webHostUrl));
        } else {
          cell.append(document.createTextNode(value ?? ''));
        }
        if (index === 2 && lease.privateMAC) {
          cell.className = 'vendor-private';
          cell.title = t('privateMACAddressTitle');
        } else if (index === 2 && !lease.vendor && validMac(lease.macAddress)) {
          cell.className = 'vendor-unknown';
          cell.title = t('vendorUnknownTitle');
        }
        row.appendChild(cell);
      });
    row.addEventListener('click', event => {
      if (event.target.closest('a')) return;
      showDetails(lease);
    });
    row.addEventListener('keydown', event => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        showDetails(lease);
      }
    });
    return row;
  }));
  updateLeaseSummary(rows.length, leases.length);
  updateRefreshMeta();
  document.querySelectorAll('#leases th').forEach(header => {
    const indicator = header.querySelector('.sort-indicator');
    if (header.dataset.key === sort.column) {
      header.setAttribute('aria-sort', sort.dir === 'asc' ? 'ascending' : 'descending');
      indicator.textContent = sort.dir === 'asc' ? ' ▲' : ' ▼';
    } else {
      indicator.textContent = '';
      header.setAttribute('aria-sort', 'none');
    }
  });
}

async function copyValue(value, button, returnLabel = t('copy')) {
  if (!value || value === t('notAvailable')) return;
  const original = button.textContent;
  const showResult = copied => {
    button.textContent = copied ? t('copied') : t('copyFailed');
    window.setTimeout(() => { if (button.isConnected) button.textContent = returnLabel; }, 1200);
  };
  try {
    await navigator.clipboard.writeText(value);
    showResult(true);
  } catch (_) {
    const area = document.createElement('textarea');
    area.value = value;
    area.style.position = 'fixed';
    area.style.opacity = '0';
    document.body.append(area);
    area.select();
    const copied = document.execCommand('copy');
    area.remove();
    showResult(copied);
  }
  if (!button.isConnected) button.textContent = original;
}

async function setKnown(lease, action, button) {
  button.disabled = true;
  button.textContent = t('saving');
  try {
    const response = await fetch(KNOWN_MACS_API_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mac: lease.macAddress, action })
    });
    if (!response.ok) throw new Error(await response.text());
    leases = leases.map(item => item.macAddress === lease.macAddress
      ? { ...item, isNew: action === 'unmark' } : item);
    render();
    refreshOpenDetails(lease.macAddress);
  } catch (error) {
    alert(t('couldNotUpdateDevice', { error: error.message }));
  } finally {
    if (button.isConnected) button.disabled = false;
    refreshOpenDetails(lease.macAddress);
  }
}

async function resetKnown() {
  if (!confirm(t('confirmReset'))) return;
  const button = document.querySelector('#reset-known-button');
  button.disabled = true;
  const label = button.querySelector('[data-i18n]');
  if (label) label.textContent = t('resetting');
  try {
    const response = await fetch(KNOWN_MACS_API_URL, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'reset' })
    });
    if (!response.ok) throw new Error(await response.text());
    leases = leases.map(lease => ({ ...lease, isNew: validMac(lease.macAddress) }));
    render();
    document.querySelector('#device-details').close();
  } catch (error) {
    alert(t('couldNotReset', { error: error.message }));
  } finally {
    button.disabled = false;
    if (label) label.textContent = t('resetRemembered');
  }
}

document.querySelector('#reset-known-button').addEventListener('click', resetKnown);
document.querySelector('#mark-reservations-button').addEventListener('click', markReservationsKnown);
const markAllIPv4ReservationsButton = document.querySelector('#mark-all-ipv4-reservations-button');
if (markAllIPv4ReservationsButton) {
  markAllIPv4ReservationsButton.addEventListener('click', markAllIPv4ReservationsKnown);
}

async function markReservationsKnown() {
  const reservedMACs = new Set(leases
    .filter(lease => lease.staticIP && validMac(lease.macAddress))
    .map(lease => String(lease.macAddress).trim().toUpperCase()));
  if (!reservedMACs.size) {
    alert(t('noReservations'));
    return;
  }
  if (!confirm(t('confirmMarkReservations', { count: reservedMACs.size }))) return;
  const button = document.querySelector('#mark-reservations-button');
  button.disabled = true;
  const label = button.querySelector('[data-i18n]');
  if (label) label.textContent = t('marking');
  try {
    const response = await fetch(KNOWN_MACS_API_URL, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'mark-reservations' })
    });
    if (!response.ok) throw new Error(await response.text());
    const result = await response.json();
    leases = leases.map(lease => reservedMACs.has(String(lease.macAddress ?? '').trim().toUpperCase())
      ? { ...lease, isNew: false } : lease);
    render();
    alert(t('markedReservations', {
      count: Number.isFinite(result.count) ? result.count : reservedMACs.size
    }));
    refreshOpenDetails(document.querySelector('#device-details').dataset.mac || '');
  } catch (error) {
    alert(t('couldNotMarkReservations', { error: error.message }));
  } finally {
    button.disabled = false;
    if (label) label.textContent = t('markReservationsKnown');
  }
}

async function markAllIPv4ReservationsKnown() {
  if (!confirm(t('confirmMarkAllIPv4Reservations'))) return;
  const button = document.querySelector('#mark-all-ipv4-reservations-button');
  if (!button) return;
  button.disabled = true;
  const label = button.querySelector('[data-i18n]');
  if (label) label.textContent = t('markingAllIPv4Reservations');
  try {
    const response = await fetch(KNOWN_MACS_API_URL, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'mark-all-ipv4-reservations' })
    });
    if (!response.ok) throw new Error(await response.text());
    const result = await response.json();
    const reservedMACs = new Set((Array.isArray(result.macs) ? result.macs : [])
      .map(mac => String(mac ?? '').trim().toUpperCase()).filter(validMac));
    if (Number(result.count) === 0 || reservedMACs.size === 0) {
      alert(t('noIPv4Reservations'));
      return;
    }
    leases = leases.map(lease => reservedMACs.has(String(lease.macAddress ?? '').trim().toUpperCase())
      ? { ...lease, isNew: false } : lease);
    render();
    alert(t('markedAllIPv4Reservations', { count: Number(result.count) }));
    refreshOpenDetails(document.querySelector('#device-details').dataset.mac || '');
  } catch (error) {
    alert(t('couldNotMarkAllIPv4Reservations', { error: error.message }));
  } finally {
    if (button.isConnected) button.disabled = false;
    if (label) label.textContent = t('markAllIPv4Reservations');
  }
}

function validMac(mac) {
  return /^(?:[0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$/.test(String(mac ?? '').trim());
}

function getDetailStatus(statusValue) {
  switch (statusValue) {
    case 'online': return t('httpHttpsGreen');
    case 'reachable': return t('pingYellow');
    case 'offline': return t('noResponseRed');
    default: return t('unknown');
  }
}

function showDetails(lease) {
  activeDetailLease = lease;
  const dialog = document.querySelector('#device-details');
  const content = document.querySelector('#detail-content');
  dialog.dataset.mac = String(lease.macAddress ?? '').trim();
  dialog.dataset.leaseKey = leaseIdentity(lease);
  content.replaceChildren();

  const details = [
    { key: 'status', value: getDetailStatus(lease.status) },
    { key: 'hostname', value: lease.name && lease.name !== '*' ? lease.name : t('notAvailable'), copy: true },
    { key: 'ipAddressV4V6', value: lease.ipAddress, copy: true },
    { key: 'macAddress', value: lease.macAddress || t('notAvailable'), copy: Boolean(lease.macAddress) },
    { key: 'vendor', value: displayVendor(lease) || t('notAvailable') },
    ...(lease.iaid !== undefined && lease.iaid !== null && String(lease.iaid) !== ''
      ? [{ key: 'iaid', value: String(lease.iaid), copy: true }] : []),
    ...(lease.duid ? [{ key: 'duid', value: lease.duid, copy: true }] : []),
    { key: 'dhcpReservation', value: lease.staticIP ? t('yes') : t('no') },
    { key: 'leaseEndsOn', value: lease.leaseExpiry === 0 ? t('never') : lease.leasetime },
    { key: 'webInterface', value: lease.webUrl || t('notDetected'), url: lease.webUrl },
    { key: 'hostnameURL', value: lease.webHostUrl || t('notDetected'), url: lease.webHostUrl }
  ];

  for (const item of details) {
    const box = document.createElement('div');
    box.className = 'detail-item';
    const label = document.createElement('span');
    label.className = 'detail-label';
    label.textContent = t(item.key);
    box.append(label);
    if (item.url) {
      box.append(link(item.value, item.url));
    } else {
      const value = document.createElement('strong');
      value.textContent = item.value ?? '';
      box.append(value);
      if (item.copy && item.value && item.value !== t('notAvailable')) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'copy-button';
        button.textContent = t('copy');
        button.title = t('copyValueTitle', { field: t(item.key).toLowerCase() });
        button.addEventListener('click', () => copyValue(item.value, button));
        box.append(button);
      }
    }
    content.append(box);
  }

  if (validMac(lease.macAddress)) {
    const action = document.createElement('div');
    action.className = 'detail-item new-device-action';
    action.id = 'new-device-action';
    const label = document.createElement('span');
    label.className = 'detail-label';
    label.textContent = lease.isNew ? t('newDevice') : t('rememberedDevice');
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'copy-button';
    button.textContent = lease.isNew ? t('markAsKnown') : t('markAsNew');
    button.addEventListener('click', () => setKnown(lease, lease.isNew ? 'mark' : 'unmark', button));
    action.append(label, button);

    // This destructive DHCP action is available only in native builds when
    // an interface is explicitly configured, and only for IPv4 lease entries.
    const releaseEnabled = document.body.dataset.leaseReleaseEnabled === 'true';
    const leaseIP = netIPv4(lease.ipAddress);
    if (releaseEnabled && leaseIP) {
      const releaseButton = document.createElement('button');
      releaseButton.type = 'button';
      releaseButton.className = 'copy-button lease-release-button';
      releaseButton.textContent = t('releaseIPv4Lease');
      releaseButton.title = t('releaseIPv4LeaseTitle');
      releaseButton.addEventListener('click', () => releaseIPv4Lease(lease, releaseButton));
      action.append(releaseButton);
    }
    content.prepend(action);
  }

  if (!dialog.open) dialog.showModal();
}

function netIPv4(value) {
  const parts = String(value ?? '').trim().split('.');
  return parts.length === 4 && parts.every(part => /^\d{1,3}$/.test(part) && Number(part) >= 0 && Number(part) <= 255)
    ? parts.map(part => String(Number(part))).join('.') : '';
}

async function releaseIPv4Lease(lease, button) {
  const ip = netIPv4(lease.ipAddress);
  if (!ip || !validMac(lease.macAddress)) return;
  if (!confirm(t('confirmReleaseIPv4Lease', { ip, mac: lease.macAddress }))) return;
  const markAsNewAfterRelease = confirm(t('confirmMarkAsNewAfterRelease'));

  button.disabled = true;
  button.textContent = t('releasingIPv4Lease');
  try {
    const response = await fetch('/release-lease', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ip, mac: lease.macAddress })
    });
    if (!response.ok) throw new Error((await response.text()).trim() || t('releaseIPv4LeaseFailed'));
    let resultMessage = t('releasedIPv4Lease');
    if (markAsNewAfterRelease) {
      const knownResponse = await fetch(KNOWN_MACS_API_URL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mac: lease.macAddress, action: 'unmark' })
      });
      if (!knownResponse.ok) {
        resultMessage += '\n\n' + t('couldNotMarkNewAfterRelease', { error: (await knownResponse.text()).trim() });
      } else {
        leases = leases.map(item => item.macAddress === lease.macAddress ? { ...item, isNew: true } : item);
      }
    }
    document.querySelector('#device-details').close();
    alert(resultMessage);
    await load();
  } catch (error) {
    alert(t('releaseIPv4LeaseFailed') + ': ' + error.message);
  } finally {
    if (button.isConnected) {
      button.disabled = false;
      button.textContent = t('releaseIPv4Lease');
    }
  }
}

function buildDeviceDetailsText(lease) {
  const hostname = lease.name && lease.name !== '*' ? lease.name : t('notAvailable');
  const fields = [
    `${t('deviceDetails')}: ${hostname}`,
    `${t('status')}: ${getDetailStatus(lease.status)}`,
    `${t('hostname')}: ${hostname}`,
    `${t('ipAddressV4V6')}: ${lease.ipAddress || t('notAvailable')}`,
    `${t('macAddress')}: ${lease.macAddress || t('notAvailable')}`,
    `${t('vendor')}: ${displayVendor(lease) || t('notAvailable')}`,
  ];
  if (lease.iaid !== undefined && lease.iaid !== null && String(lease.iaid) !== '') {
    fields.push(`${t('iaid')}: ${lease.iaid}`);
  }
  if (lease.duid) fields.push(`${t('duid')}: ${lease.duid}`);
  fields.push(`${t('dhcpReservation')}: ${lease.staticIP ? t('yes') : t('no')}`);
  fields.push(`${t('leaseEndsOn')}: ${lease.leaseExpiry === 0 ? t('never') : formatLease(lease)}`);
  fields.push(`${t('webInterface')}: ${lease.webUrl || t('notDetected')}`);
  fields.push(`${t('hostnameURL')}: ${lease.webHostUrl || t('notDetected')}`);
  if (validMac(lease.macAddress)) {
    fields.push(`${t('newDevice')}: ${lease.isNew ? t('yes') : t('no')}`);
  }
  return fields.join('\n');
}

function leaseIdentity(lease) {
  return [lease.ipAddress ?? '', lease.name ?? '', lease.iaid ?? '', lease.duid ?? ''].join('|');
}

function refreshOpenDetails(mac) {
  const dialog = document.querySelector('#device-details');
  if (!dialog.open) return;
  const normalized = String(mac ?? '').trim().toUpperCase();
  const lease = normalized
    ? leases.find(item => String(item.macAddress ?? '').trim().toUpperCase() === normalized)
    : leases.find(item => leaseIdentity(item) === (dialog.dataset.leaseKey || ''));
  if (lease) showDetails(lease);
}

document.querySelector('#copy-device-details-button').addEventListener('click', event => {
  if (activeDetailLease) {
    copyValue(buildDeviceDetailsText(activeDetailLease), event.currentTarget, t('copyDeviceDetails'));
  }
});

document.querySelector('#detail-close').addEventListener('click', () => {
  document.querySelector('#device-details').close();
});
document.querySelector('#device-details').addEventListener('click', event => {
  if (event.target === event.currentTarget) event.currentTarget.close();
});
document.querySelector('#search').addEventListener('input', event => {
  filterQuery = event.target.value.trim();
  render();
});
document.querySelector('#ip-filter').addEventListener('change', event => {
  ipFilter = event.target.value;
  render();
});
document.querySelector('#reservation-filter').addEventListener('change', event => {
  reservationFilter = event.target.value;
  render();
});
document.querySelector('#status-filter').addEventListener('change', event => {
  statusFilter = event.target.value;
  render();
});
document.querySelector('#group-filter').addEventListener('change', event => {
  groupFilter = event.target.value;
  render();
});
document.querySelector('#leases thead').addEventListener('click', event => {
  const header = event.target.closest('th');
  if (!header) return;
  const column = header.dataset.key;
  sort = sort.column === column
    ? { column, dir: sort.dir === 'asc' ? 'desc' : 'asc' }
    : { column, dir: 'asc' };
  saveSort();
  render();
});

async function load() {
  if (refreshing) return;
  refreshing = true;
  const button = document.querySelector('#refresh-button');
  const label = button.querySelector('[data-i18n]');
  button.disabled = true;
  button.setAttribute('aria-busy', 'true');
  if (label) label.textContent = t('refreshing');
  updateRefreshMeta();
  try {
    const response = await fetch(LEASES_API_URL, { cache: 'no-store' });
    if (!response.ok) throw new Error(t('httpError', { status: response.status }));
    const payload = await response.json();
    if (!payload || !Array.isArray(payload.leases)) throw new Error(t('invalidLeaseResponse'));

    // Replace the table only after a valid response arrives, preserving stale-but-useful data on failure.
    leases = payload.leases;
    lastSuccessfulRefresh = new Date();
    lastLoadError = null;
    render();
  } catch (error) {
    lastLoadError = error.message || String(error);
    // Keep the last successful lease list visible and show the refresh error separately.
    render();
    console.error(error);
  } finally {
    refreshing = false;
    button.disabled = false;
    button.removeAttribute('aria-busy');
    if (label) label.textContent = t('refresh');
    updateRefreshMeta();
  }
}

document.querySelector('#refresh-button').addEventListener('click', load);
window.addEventListener('app-language-change', () => {
  render();
  updateRefreshMeta();
  const dialog = document.querySelector('#device-details');
  if (dialog.open) refreshOpenDetails(dialog.dataset.mac || '');
});

load();
window.setInterval(() => { if (leases.length) render(); }, COUNTDOWN_RENDER_INTERVAL_MS);

// Persistent view preferences and capability-aware feature catalog.
(() => {
  const API = '/ui-settings';
  const dialog = document.querySelector('#settings-dialog');
  if (!dialog) return;
  const features = [
    { id: 'summary-total-card', label: 'Leases gesamt', desc: 'Gesamtzahl aller erkannten Leases anzeigen.', element: '#summary-cards > .summary-card:nth-child(1)', supported: true },
    { id: 'summary-online-card', label: 'Online · Web erreichbar', desc: 'Anzahl der Geräte mit erreichbarer Weboberfläche anzeigen.', element: '#summary-cards > .summary-card:nth-child(2)', supported: true },
    { id: 'summary-ping-card', label: 'Ping erreichbar', desc: 'Anzahl der per Ping erreichbaren Geräte anzeigen.', element: '#summary-cards > .summary-card:nth-child(3)', supported: true },
    { id: 'summary-offline-card', label: 'Keine Antwort', desc: 'Anzahl der Geräte ohne Antwort anzeigen.', element: '#summary-cards > .summary-card:nth-child(4)', supported: true },
    { id: 'summary-expiring-card', label: 'Läuft in 24 Stunden ab', desc: 'Anzahl der in den nächsten 24 Stunden auslaufenden Leases anzeigen.', element: '#summary-cards > .summary-card:nth-child(5)', supported: true },
    { id: 'search', label: 'Lease-Suche', desc: 'Geräte anhand der Lease-Daten filtern.', element: '#search', supported: true },
    { id: 'ip-filter', label: 'IP-Filter', desc: 'Alle IPs, IPv4 oder IPv6 anzeigen.', element: '#ip-filter', supported: true },
    { id: 'reservation-filter', label: 'Lease-Filter', desc: 'Reservierungen oder dynamische Leases auswählen.', element: '#reservation-filter', supported: true },
    { id: 'status-filter', label: 'Status-Filter', desc: 'Nach Online-, Ping- oder Offline-Status filtern.', element: '#status-filter', supported: true },
    { id: 'group-filter', label: 'Gruppenfilter', desc: 'Geräte nach Gruppe filtern.', element: '#group-filter', supported: true },
    { id: 'status-column', label: 'Statusspalte', desc: 'Status und Erreichbarkeit der Geräte anzeigen.', element: null, column: 'status', supported: true },
    { id: 'ip-column', label: 'IP-Adresse', desc: 'Die IP-Adresse anzeigen.', element: null, column: 'ip', supported: true },
    { id: 'mac-column', label: 'MAC-Adresse', desc: 'Die MAC-Adresse anzeigen.', element: null, column: 'mac', supported: true },
    { id: 'vendor-column', label: 'Hersteller', desc: 'Den ermittelten Gerätehersteller anzeigen.', element: null, column: 'vendor', supported: true },
    { id: 'name-column', label: 'Name', desc: 'Den Gerätenamen anzeigen.', element: null, column: 'name', supported: true },
    { id: 'group-column', label: 'Gruppenspalte', desc: 'Die Gruppenspalte in der Lease-Tabelle anzeigen.', element: null, column: 'group', supported: true },
    { id: 'reservation-column', label: 'DHCP-Reservierung', desc: 'Anzeigen, ob eine DHCP-Reservierung besteht.', element: null, column: 'reservation', supported: true },
    { id: 'lease-expiry-column', label: 'Lease läuft ab', desc: 'Die verbleibende Lease-Laufzeit anzeigen.', element: null, column: 'lease-expiry', supported: true },
    { id: 'dhcp-event-column', label: 'Letztes DHCP-Ereignis', desc: 'Das letzte DHCP-Ereignis anzeigen.', element: null, column: 'dhcp-event', supported: !!document.querySelector('#leases th[data-column="dhcp-event"]') },
    { id: 'mark-reservations-button', label: 'Reservierungen merken', desc: 'Erkannte DHCP-Reservierungen als bekannt markieren.', element: '#mark-reservations-button', supported: true },
    { id: 'mark-all-ipv4-reservations-button', label: 'Alle IPv4-Reservierungen merken', desc: 'Reservierungen aus der DHCP-Hosts-Datei übernehmen.', element: '#mark-all-ipv4-reservations-button', supported: true },
    { id: 'reset-known-button', label: 'Merkliste zurücksetzen', desc: 'Alle gespeicherten bekannten Geräte vergessen.', element: '#reset-known-button', supported: true },
    { id: 'theme-toggle', label: 'Design umschalten', desc: 'Zwischen hellem und dunklem Design wechseln.', element: '#theme-toggle', supported: true },
    { id: 'release-lease', label: 'IPv4-Lease freigeben', desc: 'Native Funktion; benötigt Native-Build und konfigurierte DHCP-Schnittstelle.', element: null, supported: document.body.dataset.leaseReleaseEnabled === 'true' },
    { id: 'dhcp-events', label: 'DHCP-Ereignisprotokoll', desc: 'Zeigt die letzten DHCP-Ereignisse aus der Datenbank.', element: '#dhcp-events', supported: !!document.querySelector('#dhcp-events') }
  ];
  const list = document.querySelector('#settings-feature-list');
  const visible = {};
  let settings = { language: 'de', visible: {} };
  const settingsText = {
    de: { title:'Ansicht & Einstellungen', button:'Einstellungen', description:'Wähle, welche Bedienelemente angezeigt werden. Grau markierte Funktionen werden von dieser Installation nicht unterstützt.', language:'Sprache', functions:'Buttons und Menüpunkte', save:'Speichern', saving:'Speichere…', saved:'Gespeichert', failed:'Speichern fehlgeschlagen – Schreibrechte für das Einstellungsverzeichnis prüfen.', unavailable:'In dieser Installation nicht verfügbar.', features:{'summary-total-card':['Leases gesamt','Gesamtzahl aller erkannten Leases anzeigen.'],'summary-online-card':['Online · Web erreichbar','Anzahl der Geräte mit erreichbarer Weboberfläche anzeigen.'],'summary-ping-card':['Ping erreichbar','Anzahl der per Ping erreichbaren Geräte anzeigen.'],'summary-offline-card':['Keine Antwort','Anzahl der Geräte ohne Antwort anzeigen.'],'summary-expiring-card':['Läuft in 24 Stunden ab','Anzahl der in den nächsten 24 Stunden auslaufenden Leases anzeigen.'],'search':['Lease-Suche','Geräte anhand der Lease-Daten filtern.'],'ip-filter':['IP-Filter','Alle IPs, IPv4 oder IPv6 anzeigen.'],'reservation-filter':['Lease-Filter','Reservierungen oder dynamische Leases auswählen.'],'status-filter':['Status-Filter','Nach Online-, Ping- oder Offline-Status filtern.'],'group-filter':['Gruppenfilter','Geräte nach Gruppe filtern.'],'status-column':['Statusspalte','Status und Erreichbarkeit der Geräte anzeigen.'],'ip-column':['IP-Adresse','Die IP-Adresse anzeigen.'],'mac-column':['MAC-Adresse','Die MAC-Adresse anzeigen.'],'vendor-column':['Hersteller','Den ermittelten Gerätehersteller anzeigen.'],'name-column':['Name','Den Gerätenamen anzeigen.'],'group-column':['Gruppenspalte','Die Gruppenspalte in der Lease-Tabelle anzeigen.'],'reservation-column':['DHCP-Reservierung','Anzeigen, ob eine DHCP-Reservierung besteht.'],'lease-expiry-column':['Lease läuft ab','Die verbleibende Lease-Laufzeit anzeigen.'],'dhcp-event-column':['Letztes DHCP-Ereignis','Das letzte DHCP-Ereignis anzeigen.'],'mark-reservations-button':['Reservierungen merken','Erkannte DHCP-Reservierungen als bekannt markieren.'],'mark-all-ipv4-reservations-button':['Alle IPv4-Reservierungen merken','Reservierungen aus der DHCP-Hosts-Datei übernehmen.'],'reset-known-button':['Merkliste zurücksetzen','Alle gespeicherten bekannten Geräte vergessen.'],'theme-toggle':['Design umschalten','Zwischen hellem und dunklem Design wechseln.'],'release-lease':['IPv4-Lease freigeben','Native Funktion; benötigt Native-Build und konfigurierte DHCP-Schnittstelle.'],'dhcp-events':['DHCP-Ereignisprotokoll','Benötigt eine konfigurierte DHCP-Eventdatei.']} },
    en: { title:'View & Settings', button:'Settings', description:'Choose which controls are shown. Greyed-out features are not supported by this installation.', language:'Language', functions:'Buttons and menu items', save:'Save', saving:'Saving…', saved:'Saved', failed:'Could not save settings – check permissions on the settings directory.', unavailable:'Not available in this installation.', features:{'summary-total-card':['Total leases','Show the total number of detected leases.'],'summary-online-card':['Online · web reachable','Show devices with a reachable web interface.'],'summary-ping-card':['Ping reachable','Show the number of devices reachable by ping.'],'summary-offline-card':['No response','Show devices that did not respond.'],'summary-expiring-card':['Expires within 24 hours','Show leases expiring in the next 24 hours.'],'search':['Lease search','Filter devices using lease data.'],'ip-filter':['IP filter','Show all IPs, IPv4 or IPv6.'],'reservation-filter':['Lease filter','Choose reservations or dynamic leases.'],'status-filter':['Status filter','Filter by online, ping or offline status.'],'group-filter':['Group filter','Filter devices by group.'],'status-column':['Status column','Show device status and reachability.'],'ip-column':['IP address','Show the IP address.'],'mac-column':['MAC address','Show the MAC address.'],'vendor-column':['Manufacturer','Show the detected device manufacturer.'],'name-column':['Name','Show the device name.'],'group-column':['Group column','Show the group column in the lease table.'],'reservation-column':['DHCP reservation','Show whether a DHCP reservation exists.'],'lease-expiry-column':['Lease expiry','Show the remaining lease time.'],'dhcp-event-column':['Last DHCP event','Show the last DHCP event.'],'mark-reservations-button':['Remember reservations','Mark detected DHCP reservations as known.'],'mark-all-ipv4-reservations-button':['Remember all IPv4 reservations','Import reservations from the DHCP hosts file.'],'reset-known-button':['Reset remembered list','Forget all remembered devices.'],'theme-toggle':['Toggle theme','Switch between light and dark themes.'],'release-lease':['Release IPv4 lease','Native feature; requires a native build and configured DHCP interface.'],'dhcp-events':['DHCP event log','Requires a configured DHCP event file.']} },
    es: { title:'Vista y ajustes', button:'Ajustes', description:'Elige qué controles se muestran. Las funciones en gris no son compatibles con esta instalación.', language:'Idioma', functions:'Botones y elementos de menú', save:'Guardar', saving:'Guardando…', saved:'Guardado', failed:'No se pudo guardar; comprueba los permisos del directorio de ajustes.', unavailable:'No disponible en esta instalación.', features:{'summary-total-card':['Total de leases','Mostrar el número total de leases detectados.'],'summary-online-card':['En línea · web accesible','Mostrar dispositivos con interfaz web accesible.'],'summary-ping-card':['Ping accesible','Mostrar el número de dispositivos accesibles por ping.'],'summary-offline-card':['Sin respuesta','Mostrar dispositivos que no respondieron.'],'summary-expiring-card':['Caduca en 24 horas','Mostrar leases que caducan en las próximas 24 horas.'],'search':['Buscar leases','Filtrar dispositivos por datos de leases.'],'ip-filter':['Filtro IP','Mostrar todas las IP, IPv4 o IPv6.'],'reservation-filter':['Filtro de leases','Elegir reservas o leases dinámicos.'],'status-filter':['Filtro de estado','Filtrar por estado en línea, ping o sin conexión.'],'group-filter':['Filtro de grupos','Filtrar dispositivos por grupo.'],'status-column':['Columna de estado','Mostrar estado y disponibilidad del dispositivo.'],'ip-column':['Dirección IP','Mostrar la dirección IP.'],'mac-column':['Dirección MAC','Mostrar la dirección MAC.'],'vendor-column':['Fabricante','Mostrar el fabricante detectado.'],'name-column':['Nombre','Mostrar el nombre del dispositivo.'],'group-column':['Columna de grupo','Mostrar la columna de grupo en la tabla.'],'reservation-column':['Reserva DHCP','Mostrar si existe una reserva DHCP.'],'lease-expiry-column':['Caducidad del lease','Mostrar el tiempo restante del lease.'],'dhcp-event-column':['Último evento DHCP','Mostrar el último evento DHCP.'],'mark-reservations-button':['Recordar reservas','Marcar reservas DHCP detectadas como conocidas.'],'mark-all-ipv4-reservations-button':['Recordar todas las reservas IPv4','Importar reservas del archivo DHCP hosts.'],'reset-known-button':['Restablecer lista','Olvidar todos los dispositivos recordados.'],'theme-toggle':['Cambiar tema','Alternar entre tema claro y oscuro.'],'release-lease':['Liberar lease IPv4','Función nativa; requiere compilación nativa e interfaz DHCP configurada.'],'dhcp-events':['Registro de eventos DHCP','Requiere un archivo de eventos DHCP configurado.']} }
  };
  function applySettingsLanguage() {
    const tx = settingsText[settings.language] || settingsText.de;
    document.querySelector('#settings-title').textContent = tx.title;
    const settingsButton = document.querySelector('#settings-button');
    if (settingsButton) { settingsButton.textContent = `⚙ ${tx.button}`; settingsButton.title = tx.title; settingsButton.setAttribute('aria-label', tx.title); }
    document.querySelector('#settings-description').textContent = tx.description;
    document.querySelector('#settings-language-label').textContent = tx.language;
    document.querySelector('#settings-functions-heading').textContent = tx.functions;
    document.querySelector('#settings-save').textContent = tx.save;
  }
  function applyColumnVisibility() {
    const columnFeatures = features.filter(f => f.column);
    const hiddenColumns = columnFeatures.filter(f => settings.visible[f.id] === false).map(f => f.column);
    document.body.dataset.hiddenColumns = JSON.stringify(hiddenColumns);
    document.body.dataset.showGroupColumn = String(settings.visible['group-column'] !== false);
    const groupsExist = document.body.dataset.hasGroups === 'true';
    columnFeatures.forEach(f => {
      const enabled = settings.visible[f.id] !== false;
      document.querySelectorAll(`#leases [data-column="${f.column}"]`).forEach(cell => {
        const groupUnavailable = f.column === 'group' && !groupsExist;
        cell.style.display = enabled && !groupUnavailable ? '' : 'none';
        if (cell.tagName === 'TH' && f.column === 'group') cell.hidden = groupUnavailable;
      });
    });
  }

  function applyVisibility() {
    features.forEach(f => {
      if (f.column) return;
      const el = document.querySelector(f.element);
      if (!el) return;
      const show = f.supported && settings.visible[f.id] !== false;
      // Respect the app's own hidden state (for example, the group filter is
      // only shown after groups have actually been discovered).
      // Keep grid layout tidy when toolbar controls are hidden.
      if (!show) el.style.display = 'none';
      else el.style.removeProperty('display');
    });
    applyColumnVisibility();
  }

  function renderFeatures() {
    list.replaceChildren();
    const tx = settingsText[settings.language] || settingsText.de;
    features.forEach(f => {
      const translated = tx.features[f.id] || [f.label, f.desc];
      const row = document.createElement('label');
      row.className = `settings-feature${f.supported ? '' : ' unavailable'}`;
      const checkbox = document.createElement('input');
      checkbox.type = 'checkbox';
      checkbox.checked = f.supported && settings.visible[f.id] !== false;
      checkbox.disabled = !f.supported;
      checkbox.addEventListener('change', () => { settings.visible[f.id] = checkbox.checked; });
      const text = document.createElement('span');
      const title = document.createElement('strong'); title.textContent = translated[0];
      const desc = document.createElement('small'); desc.textContent = f.supported ? translated[1] : `${translated[1]} ${tx.unavailable}`;
      text.append(title, desc); row.append(checkbox, text); list.append(row);
    });
  }

  async function loadSettings() {
    try {
      const response = await fetch(API, { cache: 'no-store' });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const loaded = await response.json();
      settings = { language: loaded.language || 'de', visible: loaded.visible || {} };
      if (window.DnsmasqI18n) window.DnsmasqI18n.setLanguage(settings.language);
    } catch (err) {
      // The UI remains usable with defaults if the settings file is read-only/unavailable.
      settings = { language: (window.DnsmasqI18n && window.DnsmasqI18n.language) || 'de', visible: {} };
    }
    document.querySelector('#settings-language').value = settings.language;
    applySettingsLanguage(); renderFeatures(); applyVisibility(); applyColumnVisibility();
  }

  window.addEventListener('app-language-change', event => {
    const language = event && event.detail && event.detail.language;
    if (!language || !settingsText[language]) return;
    settings.language = language;
    document.querySelector('#settings-language').value = language;
    applySettingsLanguage();
    renderFeatures();
  });
  document.querySelector('#settings-button').addEventListener('click', () => dialog.showModal());
  document.querySelector('#settings-close').addEventListener('click', () => dialog.close());
  document.querySelector('#settings-language').addEventListener('change', e => { settings.language = e.target.value; applySettingsLanguage(); renderFeatures(); });
  document.querySelector('#settings-save').addEventListener('click', async () => {
    const state = document.querySelector('#settings-save-state');
    settings.language = document.querySelector('#settings-language').value;
    const tx = settingsText[settings.language] || settingsText.de;
    state.textContent = tx.saving;
    try {
      const response = await fetch(API, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(settings) });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      if (window.DnsmasqI18n) window.DnsmasqI18n.setLanguage(settings.language);
      applyVisibility(); applyColumnVisibility(); state.textContent = tx.saved;
      window.setTimeout(() => { state.textContent = ''; dialog.close(); }, 500);
    } catch (_) { state.textContent = tx.failed; }
  });
  loadSettings();
})();


// DHCP event history is stored in the database; no legacy log file is required.
(() => {
  const button = document.querySelector('#dhcp-events');
  const dialog = document.querySelector('#dhcp-events-dialog');
  const close = document.querySelector('#dhcp-events-close');
  const status = document.querySelector('#dhcp-events-status');
  const body = document.querySelector('#dhcp-events-rows');
  if (!button || !dialog || !body) return;
  const cell = (row, value) => {
    const td = document.createElement('td');
    td.textContent = value == null ? '' : String(value);
    row.appendChild(td);
  };
  button.addEventListener('click', async () => {
    dialog.showModal();
    status.textContent = 'Lade Ereignisse …';
    body.replaceChildren();
    try {
      const response = await fetch('/dhcp-events');
      if (!response.ok) throw new Error('HTTP ' + response.status);
      const data = await response.json();
      for (const event of (data.events || [])) {
        const row = document.createElement('tr');
        cell(row, event.time); cell(row, event.action); cell(row, event.ip);
        cell(row, event.mac); cell(row, event.hostname);
        body.appendChild(row);
      }
      status.textContent = (data.events || []).length + ' Ereignisse geladen.';
    } catch (error) {
      status.textContent = 'Ereignisse konnten nicht geladen werden: ' + error.message;
    }
  });
  close?.addEventListener('click', () => dialog.close());
})();
