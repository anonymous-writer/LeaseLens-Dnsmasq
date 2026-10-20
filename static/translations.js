/*
 * Central UI translations for leaselens-dnsmasq.
 * Add or edit visible English/German wording here instead of in app.js/index.html.
 * Language preference is stored in localStorage and shared by this browser.
 */
(function () {
  'use strict';

  const STORAGE_KEY = 'dnsmasq-leases-language';

  const dictionaries = {
    en: {
      language: 'Language', selectLanguage: 'Select language',
      brandSubtitle: 'Dnsmasq Network Dashboard', brandDescription: 'DHCP leases · Device discovery · Manufacturers · Network status', systemTime: 'System time', systemLabel: 'System', systemDescription: 'Go · Web Dashboard', dnsmasqRunning: '● Running', dnsmasqVersion: 'Version {version}', dnsmasqMonitoring: 'Status monitored',
      summaryTotal: 'Total leases', summaryOnline: 'Online · web reachable', summaryPing: 'Ping reachable', summaryExpiring: 'Expires within 24 hours', summaryOffline: 'No response',
      searchPlaceholder: 'Filter leases…', ipFilterTitle: 'Filter IPv4/IPv6',
      allIPs: 'All IPs', ipv4: 'IPv4', ipv6: 'IPv6',
      reservationFilterTitle: 'Filter DHCP reservations', allLeases: 'All leases',
      statusFilterTitle: 'Filter by device status', allStatuses: 'All statuses',
      webOnlineFilter: 'HTTP/HTTPS responds', pingOnlyFilter: 'Ping responds, no web UI', noResponseFilter: 'No ping or web response',
      reservations: 'Reservations', dynamicOnly: 'Dynamic only',
      refresh: 'Refresh', refreshing: 'Refreshing…', loadingLeases: 'Checking DHCP leases…',
      lastUpdated: 'Last successful update: {time}', refreshFailedKeepingData: 'Refresh failed: {error}. Keeping the previously loaded data.',
      invalidLeaseResponse: 'The server returned an invalid lease list.', copyDeviceDetails: 'Copy device details',
      markReservationsKnown: 'Mark reservations known',
      markReservationsKnownTitle: 'Mark currently detected DHCP reservations as known',
      markAllIPv4Reservations: 'Mark all IPv4 reservations',
      markAllIPv4ReservationsTitle: 'Mark all configured IPv4 reservations as known, including offline devices',
      resetRemembered: 'Reset remembered', resetRememberedTitle: 'Forget all remembered devices',
      toggleTheme: 'Toggle dark mode', status: 'Status', ipAddress: 'IP Address',
      macAddress: 'MAC Address', vendor: 'Manufacturer', vendorUnknown: 'Unknown', vendorUnknownTitle: 'No matching OUI entry in the local database', privateMACAddress: 'Private / randomized MAC', privateMACAddressTitle: 'Locally administered MAC address; manufacturer cannot be reliably inferred', iaid: 'IAID', duid: 'DUID', name: 'Name', dhcpReservation: 'DHCP Reservation',
      leaseEndsOn: 'Lease ends on', loading: 'Loading…', close: 'Close',
      deviceDetails: 'Device details', released: 'released', yes: 'Yes', no: 'No',
      never: 'Never', expired: 'Expired',
      inDaysHours: 'In {days}d {hours}h', inHoursMinutes: 'In {hours}h {minutes}m',
      inMinutes: 'In {minutes}m', inSeconds: 'In {seconds}s', newBadge: 'NEW',
      httpHttpsReachable: 'HTTP/HTTPS reachable', pingReachableNoHTTP: 'Ping reachable, no HTTP/HTTPS',
      noResponse: 'No response to ping or HTTP/HTTPS', statusUnknown: 'Status unknown',
      httpHttpsGreen: 'HTTP/HTTPS reachable (green)', pingYellow: 'Ping reachable, no HTTP/HTTPS (yellow)',
      noResponseRed: 'No response to ping or HTTP/HTTPS (red)', statusShortOnline: 'Online', statusShortPing: 'Ping OK', statusShortOffline: 'Offline', unknown: 'Unknown',
      hostname: 'Hostname', ipAddressV4V6: 'IPv4 / IPv6 address', webInterface: 'Web interface',
      hostnameURL: 'Hostname URL', notAvailable: 'Not available', notDetected: 'Not detected',
      copy: 'Copy', copied: 'Copied', copyFailed: 'Copy failed', copyValueTitle: 'Copy {field}',
      newDevice: 'New device', rememberedDevice: 'Remembered device',
      markAsKnown: 'Mark as known', markAsNew: 'Mark as new', saving: 'Saving…',
      releaseIPv4Lease: 'Remove IPv4 lease', releaseIPv4LeaseTitle: 'Ask dnsmasq to release this IPv4 DHCP lease',
      confirmReleaseIPv4Lease: 'Release the DHCP lease for {ip} ({mac})? The device may request a new lease immediately.',
      confirmMarkAsNewAfterRelease: 'Also mark this device as new again?',
      couldNotMarkNewAfterRelease: 'The lease was released, but the device could not be marked as new: {error}',
      releasingIPv4Lease: 'Releasing lease…', releasedIPv4Lease: 'The DHCP release request was sent to dnsmasq.',
      releaseIPv4LeaseFailed: 'Could not release the DHCP lease',
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
      couldNotMarkReservations: 'Could not mark currently detected DHCP reservations as known: {error}',
      confirmMarkAllIPv4Reservations: 'Mark every IPv4 reservation with a valid MAC in the DHCP hosts file as known? Offline devices are included.',
      markingAllIPv4Reservations: 'Marking all IPv4 reservations…',
      noIPv4Reservations: 'No IPv4 reservations with valid MAC addresses were found in the DHCP hosts file.',
      markedAllIPv4Reservations: 'Processed {count} IPv4 reservation(s) from the DHCP hosts file as known.',
      couldNotMarkAllIPv4Reservations: 'Could not mark configured IPv4 reservations as known: {error}',
      httpError: 'HTTP {status}',
      sortAscending: 'ascending', sortDescending: 'descending'
    },
    de: {
      language: 'Sprache', selectLanguage: 'Sprache auswählen',
      brandSubtitle: 'Dnsmasq Network Dashboard', brandDescription: 'DHCP-Leases · Geräteerkennung · Hersteller · Netzwerkstatus', systemTime: 'Systemzeit', systemLabel: 'System', systemDescription: 'Go · Web-Dashboard', dnsmasqRunning: '● Läuft', dnsmasqVersion: 'Version {version}', dnsmasqMonitoring: 'Status wird überwacht',
      summaryTotal: 'Leases gesamt', summaryOnline: 'Online · Web erreichbar', summaryPing: 'Ping erreichbar', summaryExpiring: 'Läuft in 24 Stunden ab', summaryOffline: 'Keine Antwort',
      searchPlaceholder: 'Leases filtern…', ipFilterTitle: 'IPv4/IPv6 filtern',
      allIPs: 'Alle IPs', ipv4: 'IPv4', ipv6: 'IPv6',
      reservationFilterTitle: 'DHCP-Reservierungen filtern', allLeases: 'Alle Leases',
      statusFilterTitle: 'Nach Gerätestatus filtern', allStatuses: 'Alle Status',
      webOnlineFilter: 'HTTP/HTTPS antwortet', pingOnlyFilter: 'Ping antwortet, keine Weboberfläche', noResponseFilter: 'Keine Antwort auf Ping oder Web',
      reservations: 'Reservierungen', dynamicOnly: 'Nur dynamische Leases',
      refresh: 'Aktualisieren', refreshing: 'Wird aktualisiert…', loadingLeases: 'DHCP-Leases werden geprüft…',
      lastUpdated: 'Zuletzt erfolgreich aktualisiert: {time}', refreshFailedKeepingData: 'Aktualisierung fehlgeschlagen: {error}. Die bisher geladenen Daten bleiben sichtbar.',
      invalidLeaseResponse: 'Der Server hat eine ungültige Lease-Liste zurückgegeben.', copyDeviceDetails: 'Gerätedetails kopieren',
      markReservationsKnown: 'Reservierungen merken',
      markReservationsKnownTitle: 'Alle aktuell erkannten DHCP-Reservierungen als bekannt markieren',
      markAllIPv4Reservations: 'Alle IPv4-Reservierungen merken',
      markAllIPv4ReservationsTitle: 'Alle konfigurierten IPv4-Reservierungen als bekannt markieren, auch offline Geräte',
      resetRemembered: 'Merkliste zurücksetzen', resetRememberedTitle: 'Alle gemerkten Geräte vergessen',
      toggleTheme: 'Dunkelmodus umschalten', status: 'Status', ipAddress: 'IP-Adresse',
      macAddress: 'MAC-Adresse', vendor: 'Hersteller', vendorUnknown: 'Unbekannt', vendorUnknownTitle: 'Kein passender OUI-Eintrag in der lokalen Datenbank', privateMACAddress: 'Private / zufällige MAC', privateMACAddressTitle: 'Lokal vergebene MAC-Adresse; Hersteller nicht zuverlässig ermittelbar', iaid: 'IAID (DHCPv6-Kennung)', duid: 'DUID (DHCPv6-Clientkennung)', name: 'Name', dhcpReservation: 'DHCP-Reservierung',
      leaseEndsOn: 'Lease läuft ab', loading: 'Wird geladen…', close: 'Schließen',
      deviceDetails: 'Gerätedetails', released: 'veröffentlicht', yes: 'Ja', no: 'Nein',
      never: 'Nie', expired: 'Abgelaufen',
      inDaysHours: 'In {days} Tg {hours} Std', inHoursMinutes: 'In {hours} Std {minutes} Min',
      inMinutes: 'In {minutes} Min', inSeconds: 'In {seconds} Sek', newBadge: 'NEU',
      httpHttpsReachable: 'HTTP/HTTPS erreichbar', pingReachableNoHTTP: 'Ping erreichbar, kein HTTP/HTTPS',
      noResponse: 'Keine Antwort auf Ping oder HTTP/HTTPS', statusUnknown: 'Status unbekannt',
      httpHttpsGreen: 'HTTP/HTTPS erreichbar (grün)', pingYellow: 'Ping erreichbar, kein HTTP/HTTPS (gelb)',
      noResponseRed: 'Keine Antwort auf Ping oder HTTP/HTTPS (rot)', statusShortOnline: 'Online', statusShortPing: 'Ping OK', statusShortOffline: 'Offline', unknown: 'Unbekannt',
      hostname: 'Hostname', ipAddressV4V6: 'IPv4-/IPv6-Adresse', webInterface: 'Weboberfläche',
      hostnameURL: 'Hostname-URL', notAvailable: 'Nicht verfügbar', notDetected: 'Nicht erkannt',
      copy: 'Kopieren', copied: 'Kopiert', copyFailed: 'Kopieren fehlgeschlagen', copyValueTitle: '{field} kopieren',
      newDevice: 'Neues Gerät', rememberedDevice: 'Gemerktes Gerät',
      markAsKnown: 'Als bekannt markieren', markAsNew: 'Wieder als neu markieren', saving: 'Wird gespeichert…',
      releaseIPv4Lease: 'IPv4-Lease entfernen', releaseIPv4LeaseTitle: 'dnsmasq auffordern, diese IPv4-DHCP-Lease freizugeben',
      confirmReleaseIPv4Lease: 'DHCP-Lease für {ip} ({mac}) freigeben? Das Gerät kann sofort eine neue Lease anfordern.',
      confirmMarkAsNewAfterRelease: 'Soll das Gerät zusätzlich wieder als neu markiert werden?',
      couldNotMarkNewAfterRelease: 'Die Lease wurde freigegeben, aber das Gerät konnte nicht wieder als neu markiert werden: {error}',
      releasingIPv4Lease: 'Lease wird freigegeben…', releasedIPv4Lease: 'Die Freigabe-Anfrage wurde an dnsmasq gesendet.',
      releaseIPv4LeaseFailed: 'DHCP-Lease konnte nicht freigegeben werden',
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
      couldNotMarkReservations: 'Aktuell erkannte DHCP-Reservierungen konnten nicht gemerkt werden: {error}',
      confirmMarkAllIPv4Reservations: 'Alle IPv4-Reservierungen mit gültiger MAC-Adresse aus der DHCP-Hosts-Datei als bekannt markieren? Auch ausgeschaltete Geräte werden übernommen.',
      markingAllIPv4Reservations: 'Alle IPv4-Reservierungen werden gemerkt…',
      noIPv4Reservations: 'Keine IPv4-Reservierungen mit gültigen MAC-Adressen in der DHCP-Hosts-Datei gefunden.',
      markedAllIPv4Reservations: '{count} IPv4-Reservierung(en) aus der DHCP-Hosts-Datei als bekannt übernommen.',
      couldNotMarkAllIPv4Reservations: 'Konfigurierte IPv4-Reservierungen konnten nicht als bekannt markiert werden: {error}',
      httpError: 'HTTP {status}',
      sortAscending: 'ascending', sortDescending: 'descending'    },
    es: {
      language: 'Idioma', selectLanguage: 'Seleccionar idioma',
      brandSubtitle: 'Panel de red Dnsmasq', brandDescription: 'Concesiones DHCP · Dispositivos · Fabricantes · Estado de red', systemTime: 'Hora del sistema', systemLabel: 'Sistema', systemDescription: 'Go · Panel web', dnsmasqRunning: '● En ejecución', dnsmasqVersion: 'Versión {version}', dnsmasqMonitoring: 'Estado supervisado',
      summaryTotal: 'Concesiones totales', summaryOnline: 'En línea · web accesible', summaryPing: 'Ping accesible', summaryExpiring: 'Caduca en 24 horas', summaryOffline: 'Sin respuesta',
      searchPlaceholder: 'Filtrar concesiones DHCP…', ipFilterTitle: 'Filtrar IPv4/IPv6',
      allIPs: 'Todas las IP', ipv4: 'IPv4', ipv6: 'IPv6',
      reservationFilterTitle: 'Filtrar reservas DHCP', allLeases: 'Todas las concesiones',
      statusFilterTitle: 'Filtrar por estado del dispositivo', allStatuses: 'Todos los estados',
      webOnlineFilter: 'Responde HTTP/HTTPS', pingOnlyFilter: 'Responde al ping, sin interfaz web', noResponseFilter: 'Sin respuesta de ping ni web',
      reservations: 'Reservas', dynamicOnly: 'Solo concesiones dinámicas',
      refresh: 'Actualizar', refreshing: 'Actualizando…', loadingLeases: 'Comprobando concesiones DHCP…',
      lastUpdated: 'Última actualización correcta: {time}', refreshFailedKeepingData: 'Error al actualizar: {error}. Se conservan los datos anteriores.',
      invalidLeaseResponse: 'El servidor devolvió una lista de concesiones no válida.', copyDeviceDetails: 'Copiar detalles del dispositivo',
      markReservationsKnown: 'Marcar reservas como conocidas',
      markReservationsKnownTitle: 'Marcar las reservas DHCP detectadas actualmente como conocidas',
      markAllIPv4Reservations: 'Marcar todas las reservas IPv4',
      markAllIPv4ReservationsTitle: 'Marcar todas las reservas IPv4 configuradas como conocidas, incluidos los dispositivos desconectados',
      resetRemembered: 'Restablecer dispositivos recordados',
      resetRememberedTitle: 'Olvidar todos los dispositivos recordados',
      toggleTheme: 'Cambiar modo oscuro', status: 'Estado', ipAddress: 'Dirección IP',
      macAddress: 'Dirección MAC', vendor: 'Fabricante', vendorUnknown: 'Desconocido', vendorUnknownTitle: 'No hay una entrada OUI coincidente en la base de datos local', privateMACAddress: 'MAC privada / aleatoria', privateMACAddressTitle: 'Dirección MAC administrada localmente; no se puede inferir el fabricante con fiabilidad', iaid: 'IAID (identificador DHCPv6)', duid: 'DUID (identificador del cliente DHCPv6)', name: 'Nombre', dhcpReservation: 'Reserva DHCP',
      leaseEndsOn: 'Fin de la concesión', loading: 'Cargando…', close: 'Cerrar',
      deviceDetails: 'Detalles del dispositivo', released: 'publicado', yes: 'Sí', no: 'No',
      never: 'Nunca', expired: 'Caducado',
      inDaysHours: 'En {days} d {hours} h', inHoursMinutes: 'En {hours} h {minutes} min',
      inMinutes: 'En {minutes} min', inSeconds: 'En {seconds} s', newBadge: 'NUEVO',
      httpHttpsReachable: 'HTTP/HTTPS accesible', pingReachableNoHTTP: 'Ping responde, sin HTTP/HTTPS',
      noResponse: 'Sin respuesta a ping ni HTTP/HTTPS', statusUnknown: 'Estado desconocido',
      httpHttpsGreen: 'HTTP/HTTPS accesible (verde)', pingYellow: 'Ping responde, sin HTTP/HTTPS (amarillo)',
      noResponseRed: 'Sin respuesta a ping ni HTTP/HTTPS (rojo)', statusShortOnline: 'En línea', statusShortPing: 'Ping OK', statusShortOffline: 'Sin respuesta', unknown: 'Desconocido',
      hostname: 'Nombre de host', ipAddressV4V6: 'Dirección IPv4 / IPv6',
      webInterface: 'Interfaz web', hostnameURL: 'URL del nombre de host',
      notAvailable: 'No disponible', notDetected: 'No detectado',
      copy: 'Copiar', copied: 'Copiado', copyFailed: 'Error al copiar',
      copyValueTitle: 'Copiar {field}',
      newDevice: 'Dispositivo nuevo', rememberedDevice: 'Dispositivo recordado',
      markAsKnown: 'Marcar como conocido', markAsNew: 'Marcar como nuevo',
      releaseIPv4Lease: 'Eliminar lease IPv4', releaseIPv4LeaseTitle: 'Solicitar a dnsmasq que libere esta lease DHCP IPv4',
      confirmReleaseIPv4Lease: '¿Liberar la lease DHCP de {ip} ({mac})? El dispositivo puede solicitar otra inmediatamente.',
      confirmMarkAsNewAfterRelease: '¿Marcar también este dispositivo como nuevo otra vez?',
      couldNotMarkNewAfterRelease: 'La lease se liberó, pero no se pudo volver a marcar el dispositivo como nuevo: {error}',
      releasingIPv4Lease: 'Liberando lease…', releasedIPv4Lease: 'Se envió la solicitud de liberación a dnsmasq.',
      releaseIPv4LeaseFailed: 'No se pudo liberar la lease DHCP',
      saving: 'Guardando…', resetting: 'Restableciendo…', marking: 'Marcando…',
      detailsFor: 'Detalles de {device}',
      oneLeaseSummary: '1 concesión · haz clic en una fila para ver los detalles',
      manyLeasesSummary: '{count} concesiones · haz clic en una fila para ver los detalles',
      filteredSummary: '{shown} de {total} concesiones',
      failedToLoadLeases: 'No se pudieron cargar las concesiones: {error}',
      couldNotUpdateDevice: 'No se pudo actualizar el dispositivo: {error}',
      confirmReset: '¿Olvidar todos los dispositivos recordados? Los dispositivos con una dirección MAC válida volverán a marcarse como NUEVO.',
      couldNotReset: 'No se pudo restablecer la lista de dispositivos recordados: {error}',
      noReservations: 'No se encontraron reservas DHCP con direcciones MAC válidas.',
      confirmMarkReservations: '¿Marcar {count} dispositivo(s) con reserva DHCP como conocidos?',
      markedReservations: '{count} dispositivo(s) con reserva DHCP marcado(s) como conocidos.',
      couldNotMarkReservations: 'No se pudieron marcar las reservas DHCP detectadas actualmente como conocidas: {error}',
      confirmMarkAllIPv4Reservations: '¿Marcar como conocidas todas las reservas IPv4 con MAC válida del archivo DHCP hosts? También se incluyen dispositivos desconectados.',
      markingAllIPv4Reservations: 'Marcando todas las reservas IPv4…',
      noIPv4Reservations: 'No se encontraron reservas IPv4 con MAC válida en el archivo DHCP hosts.',
      markedAllIPv4Reservations: 'Se procesaron como conocidas {count} reserva(s) IPv4 del archivo DHCP hosts.',
      couldNotMarkAllIPv4Reservations: 'No se pudieron marcar como conocidas las reservas IPv4 configuradas: {error}',
      httpError: 'HTTP {status}',
      sortAscending: 'ascendente', sortDescending: 'descendente'
    }
  };

  function getBrowserLanguage() {
    const browserLanguage = String(navigator.language || 'en').toLowerCase();
    if (browserLanguage.startsWith('es')) return 'es';
    if (browserLanguage.startsWith('de')) return 'de';
    return 'en';
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
    root.querySelectorAll('[data-i18n-version]').forEach((element) => {
      const version = element.dataset.i18nVersion || '';
      element.textContent = version ? t('dnsmasqVersion', { version }) : t('dnsmasqMonitoring');
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
    get supportedLanguages() { return ['en', 'de', 'es']; }
  };

  initialize();
})();
