/*
 * dnsmasq-leases-ui frontend behavior.
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
let sort = loadSort();
let refreshing = false;
let lastLoadError = null;
const status = document.querySelector('#status');

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
  return [lease.ipAddress, lease.macAddress, lease.name, lease.leasetime, formatLease(lease),
    ...reservationTerms, lease.status].some(value => String(value ?? '').toLowerCase().includes(query));
}

function sorted(items) {
  return [...items].sort((a, b) => {
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

function statusCell(lease) {
  const cell = document.createElement('td');
  cell.className = 'device-status';
  const dot = document.createElement('span');
  dot.className = `dot ${lease.status || 'unknown'}`;
  dot.title = statusDescription(lease.status);
  dot.setAttribute('aria-label', dot.title);
  cell.append(dot);
  return cell;
}

function matchesFilters(lease) {
  const isIPv6 = String(lease.ipAddress ?? '').includes(':');
  if (ipFilter === 'ipv4' && isIPv6) return false;
  if (ipFilter === 'ipv6' && !isIPv6) return false;
  if (reservationFilter === 'reserved' && !lease.staticIP) return false;
  if (reservationFilter === 'dynamic' && lease.staticIP) return false;
  return true;
}

function updateLeaseSummary(shown, total) {
  if (lastLoadError) {
    status.textContent = t('failedToLoadLeases', { error: lastLoadError });
    status.classList.add('error');
    return;
  }
  status.classList.remove('error');
  if (shown !== total) {
    status.textContent = t('filteredSummary', { shown, total });
  } else if (total === 1) {
    status.textContent = t('oneLeaseSummary');
  } else {
    status.textContent = t('manyLeasesSummary', { count: total });
  }
}

function render() {
  const rows = sorted(leases.filter(lease => match(lease, filterQuery) && matchesFilters(lease)));
  const tbody = document.querySelector('#leases tbody');
  tbody.replaceChildren(...rows.map(lease => {
    const row = document.createElement('tr');
    row.className = `device-row${lease.isNew ? ' new-device' : ''}`;
    row.tabIndex = 0;
    row.setAttribute('aria-label', t('detailsFor', { device: lease.name || lease.ipAddress }));
    row.appendChild(statusCell(lease));
    [lease.ipAddress, lease.macAddress, lease.name, lease.staticIP ? t('yes') : t('no'), formatLease(lease)]
      .forEach((value, index) => {
        const cell = document.createElement('td');
        if (index === 1 && lease.isNew) {
          const badge = document.createElement('span');
          badge.className = 'new-badge';
          badge.textContent = t('newBadge');
          cell.append(badge, document.createTextNode(' '));
        }
        if (index === 0 && lease.webUrl) {
          cell.appendChild(link(lease.ipAddress, lease.webUrl));
        } else if (index === 2 && lease.webHostUrl) {
          cell.appendChild(link(lease.name, lease.webHostUrl));
        } else {
          cell.append(document.createTextNode(value ?? ''));
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

async function copyValue(value, button) {
  if (!value || value === t('notAvailable')) return;
  const original = button.textContent;
  const showResult = copied => {
    button.textContent = copied ? t('copied') : t('copyFailed');
    window.setTimeout(() => { button.textContent = t('copy'); }, 1200);
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
  const dialog = document.querySelector('#device-details');
  const content = document.querySelector('#detail-content');
  dialog.dataset.mac = String(lease.macAddress ?? '').trim();
  content.replaceChildren();

  const details = [
    { key: 'status', value: getDetailStatus(lease.status) },
    { key: 'hostname', value: lease.name && lease.name !== '*' ? lease.name : t('notAvailable'), copy: true },
    { key: 'ipAddressV4V6', value: lease.ipAddress, copy: true },
    { key: 'macAddress', value: lease.macAddress, copy: true },
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
    content.prepend(action);
  }

  if (!dialog.open) dialog.showModal();
}

function refreshOpenDetails(mac) {
  const dialog = document.querySelector('#device-details');
  if (!dialog.open) return;
  const normalized = String(mac ?? '').trim().toUpperCase();
  const lease = leases.find(item => String(item.macAddress ?? '').trim().toUpperCase() === normalized);
  if (lease) showDetails(lease);
}

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
  if (label) label.textContent = t('refreshing');
  try {
    const response = await fetch(LEASES_API_URL, { cache: 'no-store' });
    if (!response.ok) throw new Error(t('httpError', { status: response.status }));
    leases = (await response.json()).leases ?? [];
    lastLoadError = null;
    status.classList.remove('error');
    render();
  } catch (error) {
    lastLoadError = error.message;
    status.textContent = t('failedToLoadLeases', { error: lastLoadError });
    status.classList.add('error');
    console.error(error);
  } finally {
    refreshing = false;
    button.disabled = false;
    if (label) label.textContent = t('refresh');
  }
}


document.querySelector('#refresh-button').addEventListener('click', load);
window.addEventListener('app-language-change', () => {
  render();
  const dialog = document.querySelector('#device-details');
  if (dialog.open) refreshOpenDetails(dialog.dataset.mac || '');
});

load();
window.setInterval(() => { if (leases.length) render(); }, COUNTDOWN_RENDER_INTERVAL_MS);
