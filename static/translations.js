/*
 * Central UI translations for dnsmasq-leases-ui.
 * Add or edit visible English/German wording here instead of in app.js/index.html.
 * Language preference is stored in localStorage and shared by this browser.
 */
(function () {
  'use strict';

  const STORAGE_KEY = 'dnsmasq-leases-language';

  const dictionaries = {
    en: {
      language: 'Language', selectLanguage: 'Select language',
      searchPlaceholder: 'Filter leases…', ipFilterTitle: 'Filter IPv4/IPv6',
      allIPs: 'All IPs', ipv4: 'IPv4', ipv6: 'IPv6',
      reservationFilterTitle: 'Filter DHCP reservations', allLeases: 'All leases',
      reservations: 'Reservations', dynamicOnly: 'Dynamic only',
      refresh: 'Refresh', refreshing: 'Refreshing…',
      markReservationsKnown: 'Mark reservations known',
      markReservationsKnownTitle: 'Mark all DHCP reservations as known',
      resetRemembered: 'Reset remembered', resetRememberedTitle: 'Forget all remembered devices',
      toggleTheme: 'Toggle dark mode', status: 'Status', ipAddress: 'IP Address',
      macAddress: 'MAC Address', name: 'Name', dhcpReservation: 'DHCP Reservation',
      leaseEndsOn: 'Lease ends on', loading: 'Loading…', close: 'Close',
      deviceDetails: 'Device details', released: 'released', yes: 'Yes', no: 'No',
      never: 'Never', expired: 'Expired',
      inDaysHours: 'In {days}d {hours}h', inHoursMinutes: 'In {hours}h {minutes}m',
      inMinutes: 'In {minutes}m', inSeconds: 'In {seconds}s', newBadge: 'NEW',
      httpHttpsReachable: 'HTTP/HTTPS reachable', pingReachableNoHTTP: 'Ping reachable, no HTTP/HTTPS',
      noResponse: 'No response to ping or HTTP/HTTPS', statusUnknown: 'Status unknown',
      httpHttpsGreen: 'HTTP/HTTPS reachable (green)', pingYellow: 'Ping reachable, no HTTP/HTTPS (yellow)',
      noResponseRed: 'No response to ping or HTTP/HTTPS (red)', unknown: 'Unknown',
      hostname: 'Hostname', ipAddressV4V6: 'IPv4 / IPv6 address', webInterface: 'Web interface',
      hostnameURL: 'Hostname URL', notAvailable: 'Not available', notDetected: 'Not detected',
      copy: 'Copy', copied: 'Copied', copyFailed: 'Copy failed', copyValueTitle: 'Copy {field}',
      newDevice: 'New device', rememberedDevice: 'Remembered device',
      markAsKnown: 'Mark as known', markAsNew: 'Mark as new', saving: 'Saving…',
      resetting: 'Resetting…', marking: 'Marking…',
      detailsFor: 'Details for {device}',
      oneLeaseSummary: '1 lease · click a row for device details',
      manyLeasesSummary: '{count} leases · click a row for device details',
      filteredSummary: '{shown} of {total} leases',
      failedToLoadLeases: 'Failed to load leases: {error}',
      couldNotUpdateDevice: 'Could not update device: {error}',
      confirmReset: 'Forget all remembered devices? Devices with valid MAC addresses will be marked NEW again.',
      couldNotReset: 'Could not reset remembered devices: {error}',
      noReservations: 'No DHCP reservations with valid MAC addresses were found.',
      confirmMarkReservations: 'Mark {count} DHCP reservation device(s) as known?',
      markedReservations: 'Marked {count} DHCP reservation device(s) as known.',
      couldNotMarkReservations: 'Could not mark DHCP reservations as known: {error}',
      httpError: 'HTTP {status}',
      sortAscending: 'ascending', sortDescending: 'descending'
    },
    de: {
      language: 'Sprache', selectLanguage: 'Sprache auswählen',
      searchPlaceholder: 'Leases filtern…', ipFilterTitle: 'IPv4/IPv6 filtern',
      allIPs: 'Alle IPs', ipv4: 'IPv4', ipv6: 'IPv6',
      reservationFilterTitle: 'DHCP-Reservierungen filtern', allLeases: 'Alle Leases',
      reservations: 'Reservierungen', dynamicOnly: 'Nur dynamische Leases',
      refresh: 'Aktualisieren', refreshing: 'Wird aktualisiert…',
      markReservationsKnown: 'Reservierungen merken',
      markReservationsKnownTitle: 'Alle DHCP-Reservierungen als bekannt markieren',
      resetRemembered: 'Merkliste zurücksetzen', resetRememberedTitle: 'Alle gemerkten Geräte vergessen',
      toggleTheme: 'Dunkelmodus umschalten', status: 'Status', ipAddress: 'IP-Adresse',
      macAddress: 'MAC-Adresse', name: 'Name', dhcpReservation: 'DHCP-Reservierung',
      leaseEndsOn: 'Lease läuft ab', loading: 'Wird geladen…', close: 'Schließen',
      deviceDetails: 'Gerätedetails', released: 'veröffentlicht', yes: 'Ja', no: 'Nein',
      never: 'Nie', expired: 'Abgelaufen',
      inDaysHours: 'In {days} Tg {hours} Std', inHoursMinutes: 'In {hours} Std {minutes} Min',
      inMinutes: 'In {minutes} Min', inSeconds: 'In {seconds} Sek', newBadge: 'NEU',
      httpHttpsReachable: 'HTTP/HTTPS erreichbar', pingReachableNoHTTP: 'Ping erreichbar, kein HTTP/HTTPS',
      noResponse: 'Keine Antwort auf Ping oder HTTP/HTTPS', statusUnknown: 'Status unbekannt',
      httpHttpsGreen: 'HTTP/HTTPS erreichbar (grün)', pingYellow: 'Ping erreichbar, kein HTTP/HTTPS (gelb)',
      noResponseRed: 'Keine Antwort auf Ping oder HTTP/HTTPS (rot)', unknown: 'Unbekannt',
      hostname: 'Hostname', ipAddressV4V6: 'IPv4-/IPv6-Adresse', webInterface: 'Weboberfläche',
      hostnameURL: 'Hostname-URL', notAvailable: 'Nicht verfügbar', notDetected: 'Nicht erkannt',
      copy: 'Kopieren', copied: 'Kopiert', copyFailed: 'Kopieren fehlgeschlagen', copyValueTitle: '{field} kopieren',
      newDevice: 'Neues Gerät', rememberedDevice: 'Gemerktes Gerät',
      markAsKnown: 'Als bekannt markieren', markAsNew: 'Wieder als neu markieren', saving: 'Wird gespeichert…',
      resetting: 'Wird zurückgesetzt…', marking: 'Wird gemerkt…',
      detailsFor: 'Details für {device}',
      oneLeaseSummary: '1 Lease · Zeile anklicken, um Gerätedetails zu öffnen',
      manyLeasesSummary: '{count} Leases · Zeile anklicken, um Gerätedetails zu öffnen',
      filteredSummary: '{shown} von {total} Leases',
      failedToLoadLeases: 'Leases konnten nicht geladen werden: {error}',
      couldNotUpdateDevice: 'Gerät konnte nicht aktualisiert werden: {error}',
      confirmReset: 'Alle gemerkten Geräte vergessen? Geräte mit gültiger MAC-Adresse werden wieder als NEU markiert.',
      couldNotReset: 'Merkliste konnte nicht zurückgesetzt werden: {error}',
      noReservations: 'Keine DHCP-Reservierungen mit gültigen MAC-Adressen gefunden.',
      confirmMarkReservations: '{count} DHCP-Reservierung(en) als bekannt markieren?',
      markedReservations: '{count} DHCP-Reservierung(en) als bekannt markiert.',
      couldNotMarkReservations: 'DHCP-Reservierungen konnten nicht gemerkt werden: {error}',
      httpError: 'HTTP {status}',
      sortAscending: 'ascending', sortDescending: 'descending'
    }
  };

  function getBrowserLanguage() {
    return String(navigator.language || 'en').toLowerCase().startsWith('de') ? 'de' : 'en';
  }

  function readSavedLanguage() {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved && Object.prototype.hasOwnProperty.call(dictionaries, saved)) return saved;
    } catch (_) { /* localStorage may be unavailable in restricted contexts */ }
    return getBrowserLanguage();
  }

  let currentLanguage = readSavedLanguage();

  function t(key, variables) {
    const dictionary = dictionaries[currentLanguage] || dictionaries.en;
    let value = dictionary[key] ?? dictionaries.en[key] ?? key;
    if (variables) {
      value = String(value).replace(/\{([a-zA-Z0-9_]+)\}/g, (match, name) =>
        Object.prototype.hasOwnProperty.call(variables, name) ? String(variables[name]) : match
      );
    }
    return String(value);
  }

  function applyStaticTranslations(root = document) {
    root.querySelectorAll('[data-i18n]').forEach((element) => {
      element.textContent = t(element.dataset.i18n);
    });
    root.querySelectorAll('[data-i18n-placeholder]').forEach((element) => {
      element.setAttribute('placeholder', t(element.dataset.i18nPlaceholder));
    });
    root.querySelectorAll('[data-i18n-title]').forEach((element) => {
      element.setAttribute('title', t(element.dataset.i18nTitle));
    });
    root.querySelectorAll('[data-i18n-aria-label]').forEach((element) => {
      element.setAttribute('aria-label', t(element.dataset.i18nAriaLabel));
    });
  }

  function setLanguage(language) {
    if (!Object.prototype.hasOwnProperty.call(dictionaries, language)) language = 'en';
    currentLanguage = language;
    try { localStorage.setItem(STORAGE_KEY, currentLanguage); } catch (_) { /* optional persistence */ }
    document.documentElement.lang = currentLanguage;
    const selector = document.querySelector('#language-select');
    if (selector) selector.value = currentLanguage;
    applyStaticTranslations();
    window.dispatchEvent(new CustomEvent('app-language-change', { detail: { language: currentLanguage } }));
  }

  function initialize() {
    document.documentElement.lang = currentLanguage;
    const selector = document.querySelector('#language-select');
    if (selector) {
      selector.value = currentLanguage;
      selector.addEventListener('change', () => setLanguage(selector.value));
    }
    applyStaticTranslations();
  }

  window.DnsmasqI18n = {
    t,
    setLanguage,
    applyStaticTranslations,
    get language() { return currentLanguage; },
    get supportedLanguages() { return ['en', 'de']; }
  };

  initialize();
})();
