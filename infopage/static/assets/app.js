// Traefix Infopage: shows what the browser and its connection reveal, from the request the server saw
// (headers added by Traefix, embedded in the page) and from the browser's own APIs. All values are
// written with textContent, never as HTML.
(function () {
  'use strict';

  var data = {};
  try {
    data = JSON.parse(document.getElementById('infopage-data').textContent);
  } catch (e) {
    data = { headers: {} };
  }

  var prefix = (data.prefix || 'X-Traefix-').toLowerCase();
  var headers = {};
  Object.keys(data.headers || {}).forEach(function (k) {
    headers[k.toLowerCase()] = { name: k, value: data.headers[k].join(', ') };
  });

  function hdr(name) {
    var h = headers[name.toLowerCase()];
    return h ? h.value : undefined;
  }

  function tx(name) {
    return hdr(prefix + name);
  }

  // key=value pairs separated by commas, the format of X-Traefix-Tls, -Transport, -Timing, -H2-Detail, -H3.
  function parseKV(s) {
    if (!s) return null;
    var out = {};
    var parts = s.split(',');
    for (var i = 0; i < parts.length; i++) {
      var j = parts[i].indexOf('=');
      if (j <= 0) return null;
      out[parts[i].slice(0, j)] = parts[i].slice(j + 1);
    }
    return out;
  }

  // hostOf strips the port of a Go RemoteAddr: "1.2.3.4:5", "[::1]:5".
  function hostOf(addr) {
    if (!addr) return undefined;
    if (addr.charAt(0) === '[') return addr.slice(1, addr.indexOf(']'));
    var i = addr.lastIndexOf(':');
    return i > 0 && addr.indexOf(':') === i ? addr.slice(0, i) : addr;
  }

  function pct(s) {
    try { return decodeURIComponent(s); } catch (e) { return s; }
  }

  var tls = parseKV(tx('Tls')) || {};
  var transport = parseKV(tx('Transport')) || {};
  var timing = parseKV(tx('Timing')) || {};

  // ---------- DOM helpers ----------

  function el(tag, attrs, children) {
    var e = document.createElement(tag);
    if (attrs) {
      Object.keys(attrs).forEach(function (k) {
        if (k === 'text') e.textContent = attrs[k];
        else e.setAttribute(k, attrs[k]);
      });
    }
    (children || []).forEach(function (c) {
      if (c == null) return;
      e.appendChild(typeof c === 'string' ? document.createTextNode(c) : c);
    });
    return e;
  }

  function card(title, intro, body) {
    return el('div', { 'class': 'card' }, [el('h2', { text: title }), intro ? el('p', { text: intro }) : null].concat(body));
  }

  function dl(rows) {
    var d = el('dl');
    rows.forEach(function (r) {
      if (r[1] === undefined || r[1] === null || r[1] === '') return;
      d.appendChild(el('dt', { text: r[0] }));
      d.appendChild(el('dd', null, [typeof r[1] === 'string' || typeof r[1] === 'number' ? String(r[1]) : r[1]]));
    });
    return d;
  }

  function valueCell(v) {
    if (v && typeof v === 'object') return kvTable(v);
    var s = String(v);
    var kv = parseKV(s);
    if (kv && Object.keys(kv).length > 1) return kvTable(kv);
    return document.createTextNode(s);
  }

  function kvTable(obj) {
    var t = el('table');
    Object.keys(obj).forEach(function (k) {
      var v = obj[k];
      if (v === undefined || v === null || v === '') return;
      t.appendChild(el('tr', null, [el('th', { text: k }), el('td', null, [valueCell(v)])]));
    });
    return t;
  }

  function badge(text, cls) {
    return el('span', { 'class': 'badge ' + (cls || ''), text: text });
  }

  function ms(us) {
    var n = Number(us);
    if (!isFinite(n) || us === undefined || us === '') return undefined;
    return (n / 1000).toFixed(n < 10000 ? 1 : 0) + ' ms';
  }

  // ---------- browser collection ----------

  function mq(q) {
    try { return window.matchMedia(q).matches; } catch (e) { return false; }
  }

  function firstMatch(list) {
    for (var i = 0; i < list.length; i++) if (mq(list[i][0])) return list[i][1];
    return undefined;
  }

  function webgl() {
    try {
      var c = document.createElement('canvas');
      var gl = c.getContext('webgl2') || c.getContext('webgl');
      if (!gl) return null;
      var info = gl.getExtension('WEBGL_debug_renderer_info');
      return {
        version: gl.getParameter(gl.VERSION),
        vendor: info ? gl.getParameter(info.UNMASKED_VENDOR_WEBGL) : gl.getParameter(gl.VENDOR),
        renderer: info ? gl.getParameter(info.UNMASKED_RENDERER_WEBGL) : gl.getParameter(gl.RENDERER),
        shadingLanguage: gl.getParameter(gl.SHADING_LANGUAGE_VERSION),
        maxTextureSize: gl.getParameter(gl.MAX_TEXTURE_SIZE)
      };
    } catch (e) {
      return null;
    }
  }

  function navTiming() {
    try {
      var n = performance.getEntriesByType('navigation')[0];
      if (!n) return null;
      var r = function (x) { return Math.round(x * 10) / 10; };
      return {
        nextHopProtocol: n.nextHopProtocol,
        dnsMs: r(n.domainLookupEnd - n.domainLookupStart),
        connectMs: r(n.connectEnd - n.connectStart),
        tlsMs: n.secureConnectionStart > 0 ? r(n.connectEnd - n.secureConnectionStart) : 0,
        requestToFirstByteMs: r(n.responseStart - n.requestStart),
        transferSize: n.transferSize,
        type: n.type
      };
    } catch (e) {
      return null;
    }
  }

  function collect() {
    var nav = navigator;
    var conn = nav.connection || nav.mozConnection || nav.webkitConnection;
    var ro = Intl.DateTimeFormat().resolvedOptions();
    var b = {
      navigator: {
        userAgent: nav.userAgent,
        platform: nav.platform,
        vendor: nav.vendor,
        language: nav.language,
        languages: (nav.languages || []).join(', '),
        cookieEnabled: nav.cookieEnabled,
        doNotTrack: nav.doNotTrack,
        globalPrivacyControl: nav.globalPrivacyControl,
        hardwareConcurrency: nav.hardwareConcurrency,
        deviceMemory: nav.deviceMemory,
        maxTouchPoints: nav.maxTouchPoints,
        webdriver: nav.webdriver,
        pdfViewerEnabled: nav.pdfViewerEnabled,
        onLine: nav.onLine
      },
      intl: {
        timeZone: ro.timeZone,
        locale: ro.locale,
        calendar: ro.calendar,
        numberingSystem: ro.numberingSystem,
        utcOffsetMinutes: -new Date().getTimezoneOffset()
      },
      screen: {
        width: screen.width,
        height: screen.height,
        availWidth: screen.availWidth,
        availHeight: screen.availHeight,
        colorDepth: screen.colorDepth,
        devicePixelRatio: window.devicePixelRatio,
        orientation: screen.orientation ? screen.orientation.type : undefined,
        viewport: window.innerWidth + 'x' + window.innerHeight,
        outer: window.outerWidth + 'x' + window.outerHeight
      },
      media: {
        colorScheme: firstMatch([['(prefers-color-scheme: dark)', 'dark'], ['(prefers-color-scheme: light)', 'light']]),
        reducedMotion: mq('(prefers-reduced-motion: reduce)'),
        contrast: firstMatch([['(prefers-contrast: more)', 'more'], ['(prefers-contrast: less)', 'less']]) || 'no-preference',
        forcedColors: mq('(forced-colors: active)'),
        pointer: firstMatch([['(pointer: fine)', 'fine'], ['(pointer: coarse)', 'coarse'], ['(pointer: none)', 'none']]),
        hover: mq('(hover: hover)'),
        colorGamut: firstMatch([['(color-gamut: rec2020)', 'rec2020'], ['(color-gamut: p3)', 'p3'], ['(color-gamut: srgb)', 'srgb']]),
        dynamicRange: mq('(dynamic-range: high)') ? 'high' : 'standard'
      },
      network: conn ? {
        effectiveType: conn.effectiveType,
        type: conn.type,
        downlinkMbps: conn.downlink,
        rttMs: conn.rtt,
        saveData: conn.saveData
      } : null,
      navigation: navTiming(),
      webgl: webgl(),
      features: {
        secureContext: window.isSecureContext,
        serviceWorker: 'serviceWorker' in nav,
        webGPU: 'gpu' in nav,
        webAuthn: typeof window.PublicKeyCredential !== 'undefined',
        webAssembly: typeof WebAssembly === 'object',
        bluetooth: 'bluetooth' in nav,
        usb: 'usb' in nav,
        clientHints: 'userAgentData' in nav
      },
      clock: {
        browserTime: new Date().toISOString(),
        serverTime: data.time,
        skewMs: data.time ? Date.now() - Date.parse(data.time) : undefined
      }
    };

    var jobs = [];
    if (nav.userAgentData) {
      b.userAgentData = {
        brands: nav.userAgentData.brands.map(function (x) { return x.brand + ' ' + x.version; }).join(', '),
        mobile: nav.userAgentData.mobile,
        platform: nav.userAgentData.platform
      };
      if (nav.userAgentData.getHighEntropyValues) {
        jobs.push(nav.userAgentData.getHighEntropyValues(['architecture', 'bitness', 'model', 'platformVersion', 'fullVersionList', 'wow64', 'formFactors'])
          .then(function (v) {
            b.userAgentData.architecture = v.architecture;
            b.userAgentData.bitness = v.bitness;
            b.userAgentData.model = v.model;
            b.userAgentData.platformVersion = v.platformVersion;
            b.userAgentData.wow64 = v.wow64;
            b.userAgentData.formFactors = (v.formFactors || []).join(', ');
            b.userAgentData.fullVersionList = (v.fullVersionList || []).map(function (x) { return x.brand + ' ' + x.version; }).join(', ');
          }).catch(function () {}));
      }
    }
    if (nav.storage && nav.storage.estimate) {
      jobs.push(nav.storage.estimate().then(function (s) {
        b.storage = { quotaMB: Math.round(s.quota / 1048576), usageMB: Math.round(s.usage / 1048576) };
      }).catch(function () {}));
    }

    // A slow or stuck browser API (storage estimate, high entropy hints) must not leave the page blank.
    var timeout = new Promise(function (resolve) { setTimeout(resolve, 1500); });
    return Promise.race([Promise.all(jobs), timeout]).then(function () { return b; });
  }

  // ---------- interpretation ----------

  function browserName(b) {
    var ua = b.navigator.userAgent || '';
    var list = b.userAgentData && b.userAgentData.fullVersionList;
    var m;
    if ((m = ua.match(/Edg(?:A|iOS)?\/([\d.]+)/))) return 'Microsoft Edge ' + m[1].split('.')[0];
    if ((m = ua.match(/OPR\/([\d.]+)/))) return 'Opera ' + m[1].split('.')[0];
    if ((m = ua.match(/SamsungBrowser\/([\d.]+)/))) return 'Samsung Internet ' + m[1].split('.')[0];
    if ((m = ua.match(/(?:Firefox|FxiOS)\/([\d.]+)/))) return 'Firefox ' + m[1].split('.')[0];
    if ((m = ua.match(/CriOS\/([\d.]+)/))) return 'Chrome ' + m[1].split('.')[0] + ' (iOS)';
    if (list && /Brave/.test(list)) return 'Brave';
    if ((m = ua.match(/Chrome\/([\d.]+)/))) return 'Chrome ' + m[1].split('.')[0];
    if ((m = ua.match(/Version\/([\d.]+).*Safari/))) return 'Safari ' + m[1];
    return 'Desconhecido';
  }

  function osName(b) {
    var ua = b.navigator.userAgent || '';
    var ch = b.userAgentData || {};
    var m;
    if (/Windows NT 10/.test(ua)) {
      if (ch.platformVersion) return parseInt(ch.platformVersion, 10) >= 13 ? 'Windows 11' : 'Windows 10';
      return 'Windows 10 ou 11';
    }
    if (/Windows/.test(ua)) return 'Windows';
    if ((m = ua.match(/(?:iPhone|CPU) OS (\d+)[_\d]*/))) return (/iPad/.test(ua) ? 'iPadOS ' : 'iOS ') + m[1];
    if ((m = ua.match(/Android ([\d.]+)/))) return 'Android ' + m[1];
    if (/CrOS/.test(ua)) return 'ChromeOS';
    if (/Mac OS X/.test(ua)) {
      if (b.navigator.maxTouchPoints > 1) return 'iPadOS (modo desktop)';
      return ch.platformVersion ? 'macOS ' + ch.platformVersion.split('.').slice(0, 2).join('.') : 'macOS';
    }
    if (/Linux/.test(ua)) return 'Linux';
    return 'Desconhecido';
  }

  // deviceKind uses the Client Hints when the browser has them (Chromium), else the User-Agent, as Firefox
  // and Safari have no navigator.userAgentData.
  function deviceKind(b) {
    var ch = b.userAgentData;
    if (ch && ch.model) return ch.model;
    if (ch) return ch.mobile ? 'Celular' : 'Computador';
    var ua = b.navigator.userAgent || '';
    if (/Mobi|iPhone/.test(ua)) return 'Celular';
    if (/iPad|Tablet|Android/.test(ua) || (/Mac OS X/.test(ua) && b.navigator.maxTouchPoints > 1)) return 'Tablet';
    return 'Computador';
  }

  function osFamily(name) {
    if (/Windows/.test(name)) return 'windows';
    if (/Desconhecido/.test(name)) return '';
    return 'unix';
  }

  function protoName() {
    var p = tx('Proto');
    if (p === 'h3') return 'HTTP/3 (QUIC)';
    if (p === 'h2') return 'HTTP/2';
    if (p === 'h1') return 'HTTP/1.1';
    return data.proto || '?';
  }

  function ttlOS(ttlInit) {
    switch (String(ttlInit)) {
      case '64': return 'Linux, Android, macOS ou iOS';
      case '128': return 'Windows';
      case '255': return 'equipamento de rede ou Unix';
      case '32': return 'sistema antigo';
    }
    return undefined;
  }

  var LABELS = { ok: 'OK', info: 'Info', warn: 'Atenção', bad: 'Alerta' };

  function num(v) {
    var n = Number(v);
    return v === undefined || v === '' || !isFinite(n) ? undefined : n;
  }

  // pathRTT is the round trip of the path: the minimum or the handshake sample, not the smoothed RTT, which
  // includes the time the client takes to check the certificate.
  function pathRTT() {
    var c = [transport.rtt_min_us, timing.tcp_rtt_us, transport.rtt_syn_us, timing.quic_min_rtt_us, timing.quic_rtt_us, transport.rtt_us];
    for (var i = 0; i < c.length; i++) if (num(c[i]) > 0) return num(c[i]);
    return undefined;
  }

  // ---------- estimated MTU ----------

  // MTU ruler (infopage/CRITERIA.md, "MTU estimado"): 1500 is the top grade, the PPPoE range (1480 to 1492)
  // loses a little, and below it the grade falls linearly to zero at 1340, the critical minimum.
  var MTU_RULER = [[1340, 0], [1480, 90], [1492, 95], [1500, 100]];
  // Ticks of the ruler; only the spaced ones carry a label, so they do not overlap on a phone.
  var MTU_MARKS = [[1340, 'crítico', true], [1400, 'IPsec', true], [1420, 'WireGuard'], [1480, 'PPPoE com VLAN'], [1492, 'PPPoE'], [1500, 'Ethernet', true]];

  function mtuGrade(mtu) {
    var p = MTU_RULER;
    if (mtu <= p[0][0]) return 0;
    if (mtu >= p[p.length - 1][0]) return 100;
    for (var i = 1; i < p.length; i++) {
      if (mtu <= p[i][0]) {
        var a = p[i - 1], c = p[i];
        return Math.round(a[1] + (c[1] - a[1]) * (mtu - a[0]) / (c[0] - a[0]));
      }
    }
    return 100;
  }

  function mtuClass(g) {
    return g >= 95 ? 'ok' : g >= 60 ? 'info' : g >= 30 ? 'warn' : 'bad';
  }

  function mtuReading(mtu) {
    if (mtu > 1500) return 'Acima de 1500: jumbo frames ou ajuste manual.';
    if (mtu === 1500) return 'Ethernet/Wi-Fi sem túnel: o máximo da internet.';
    if (mtu >= 1480) return 'Faixa do PPPoE (fibra e DSL) ou de um túnel leve: diferença desprezível.';
    if (mtu >= 1400) return 'Túnel ou VPN no caminho (WireGuard 1420, IPsec ~1400), ou rede móvel.';
    if (mtu > 1340) return 'Muito encapsulamento no caminho: túneis empilhados, VPN sobre VPN ou ajuste agressivo.';
    return 'No mínimo crítico ou abaixo: pacotes grandes podem se perder (blackhole de PMTU).';
  }

  // quicMaxUDPPayload reads max_udp_payload from the QUIC transport parameters in X-Traefix-H3 (tp=a:1-b:2).
  // It returns null when the Traefix sent no transport parameters, and undefined when the client left the
  // parameter out.
  function quicMaxUDPPayload() {
    var h3 = parseKV(tx('H3')) || {};
    if (!h3.tp) return null;
    var m = /(?:^|-)max_udp_payload:(\d+)/.exec(h3.tp);
    return m ? Number(m[1]) : undefined;
  }

  // mtuEstimate is the MTU the client sized its packets for: the MSS of the SYN plus 20 bytes of TCP, or
  // max_udp_payload of QUIC plus 8 bytes of UDP, plus the IP header (20 bytes in IPv4, 40 in IPv6).
  function mtuEstimate() {
    var v6 = transport.ipv === '6';
    var o = { ipv: v6 ? 6 : 4, ip: v6 ? 40 : 20 };
    var mss = num(transport.syn_mss);
    if (transport.proto !== 'udp' && mss !== undefined) {
      o.base = 'MSS'; o.payload = mss; o.l4Name = 'TCP'; o.l4 = 20; o.source = 'opção MSS do SYN';
    } else if (transport.proto === 'udp') {
      var p = quicMaxUDPPayload();
      o.l4Name = 'UDP'; o.l4 = 8; o.initial = num(transport.initial_size);
      // 65527 is the default of RFC 9000: the client set no limit, so there is no MTU to read from it.
      if (p === null) {
        o.note = 'O Traefix não enviou os parâmetros de transporte do QUIC (' + data.prefix + 'H3 com tp=) nesta conexão: sem dado para estimar o MTU.' +
          (o.initial ? ' O primeiro datagrama Initial teve ' + o.initial + ' bytes de UDP, tamanho fixo do cliente, não do caminho.' : '');
        return o;
      }
      if (p !== undefined && p < 65527) {
        o.base = 'max_udp_payload'; o.payload = p; o.source = 'parâmetro de transporte max_udp_payload do QUIC';
      } else {
        o.note = 'O cliente QUIC não declarou max_udp_payload (padrão 65527): não há MTU planejado para ler.' +
          (o.initial ? ' O primeiro datagrama Initial teve ' + o.initial + ' bytes de UDP (' + (o.initial + 8 + o.ip) + ' bytes no IP), tamanho fixo do cliente, não do caminho.' : '');
        return o;
      }
    } else {
      return null;
    }
    o.mtu = o.payload + o.l4 + o.ip;
    o.grade = mtuGrade(o.mtu);
    return o;
  }

  function mtuRuler(mtu) {
    var lo = 1300, hi = 1520;
    var pos = function (v) { return Math.round(100 * (Math.min(Math.max(v, lo), hi) - lo) / (hi - lo) * 10) / 10 + '%'; };
    // The CSP forbids style attributes; the CSSOM is allowed.
    var track = el('div', { 'class': 'ruler-track' });
    var crit = el('span', { 'class': 'ruler-seg crit' });
    crit.style.left = '0';
    crit.style.width = pos(MTU_RULER[0][0]);
    track.appendChild(crit);
    MTU_RULER.slice(1).forEach(function (p, i) {
      var seg = el('span', { 'class': 'ruler-seg seg' + i });
      seg.style.left = pos(MTU_RULER[i][0]);
      seg.style.width = 'calc(' + pos(p[0]) + ' - ' + pos(MTU_RULER[i][0]) + ')';
      track.appendChild(seg);
    });
    var marks = el('div', { 'class': 'ruler-marks' });
    MTU_MARKS.forEach(function (m) {
      var tick = el('span', { 'class': 'ruler-tick', text: m[2] ? String(m[0]) : '' });
      tick.style.left = pos(m[0]);
      tick.setAttribute('title', m[1]);
      marks.appendChild(tick);
    });
    var wrap = el('div', { 'class': 'ruler' }, [track, marks]);
    if (mtu !== undefined) {
      var mk = el('span', { 'class': 'ruler-marker ' + mtuClass(mtuGrade(mtu)), text: '▲ ' + mtu });
      mk.style.left = pos(mtu);
      wrap.appendChild(mk);
    }
    return wrap;
  }

  function mtuCard() {
    var mt = mtuEstimate();
    if (!mt) return null;
    var legend = el('table', { 'class': 'eval' }, [
      [1500, 'Ethernet/Wi-Fi sem túnel'], [1492, 'PPPoE (fibra e DSL)'], [1480, 'PPPoE com VLAN, túnel leve'],
      [1420, 'WireGuard'], [1400, 'IPsec, rede móvel'], [1340, 'Mínimo crítico']
    ].map(function (r) {
      var g = mtuGrade(r[0]);
      return el('tr', null, [el('td', { 'class': 'eval-badge' }, [badge(String(g), mtuClass(g))]), el('th', { text: r[0] + ' bytes' }), el('td', { 'class': 'muted', text: r[1] })]);
    }));
    var intro = 'O MTU para o qual o seu sistema dimensionou os pacotes, pelo que ele anunciou ao abrir a conexão. Mede a rede de acesso (túneis, PPPoE, VPN), não a velocidade, e não entra na pontuação de automação.';
    if (mt.mtu === undefined) {
      return card('MTU estimado', intro, [el('p', { 'class': 'muted', text: mt.note }), mtuRuler(), legend]);
    }
    var calc = dl([
      ['Base', mt.base + ' = ' + mt.payload + ' bytes (' + mt.source + ')'],
      ['Cabeçalho ' + mt.l4Name, '+ ' + mt.l4 + ' bytes'],
      ['Cabeçalho IPv' + mt.ipv, '+ ' + mt.ip + ' bytes'],
      ['MTU estimado', el('strong', { text: mt.mtu + ' bytes' })],
      ['Nota', el('span', null, [badge(mt.grade + '/100', mtuClass(mt.grade)), ' ' + mtuReading(mt.mtu)])],
      ['Datagrama Initial', mt.initial ? mt.initial + ' bytes de UDP (tamanho fixo do cliente, não do caminho)' : undefined],
      ['MTU do servidor (PMTU)', transport.pmtu ? transport.pmtu + ' bytes (rota do servidor até você, não o seu MTU)' : undefined]
    ]);
    return card('MTU estimado', intro, [calc, mtuRuler(mt.mtu), legend]);
  }

  // ---------- connection timeline ----------

  // Same thresholds as TIME-RELAY in score.go (measures of finger-sniffer, task 313).
  var RELAY = { gap: 10000, frac: 0.3, first: 5000 };

  // relayVerdict reads the timing of the connection: through a proxy or SOCKS, the TLS round trip exceeds the TCP
  // one (the TCP ends at the proxy) and the ClientHello comes one proxy round trip after the ACK.
  function relayVerdict() {
    if (transport.proto === 'udp' || timing.proto === 'udp') {
      return { cls: 'info', state: 'na', title: 'Não aplicável', text: 'HTTP/3 (QUIC) não tem handshake TCP separado do TLS: a comparação de tempos não se aplica.' };
    }
    var gap = num(timing.rtt_gap_us), rtt = num(timing.tcp_rtt_us), est = num(timing.est_to_first_us);
    if (gap === undefined || rtt === undefined) return null;
    var relay = data.score && data.score.network && data.score.network.relay;
    var slow = gap > RELAY.gap && gap > RELAY.frac * rtt;
    if (!slow) return { cls: 'ok', state: 'direct', title: 'Conexão direta', text: 'Sem sinal de proxy ou SOCKS: o TLS levou o mesmo que o TCP' + (est !== undefined ? ' e o ClientHello chegou ' + msUs(est) + ' após o ACK.' : '.') };
    if (relay) return { cls: 'info', state: 'relay', title: 'Relay de privacidade', text: 'Atraso do ' + relay + ' (relay de privacidade da Apple): não conta como proxy.' };
    if (est !== undefined && est > RELAY.first) {
      return { cls: 'bad', state: 'proxy', title: 'Proxy ou SOCKS no caminho', text: 'Proxy ou SOCKS no caminho: o TLS levou ' + msUs(gap) + ' a mais que o TCP e o ClientHello chegou ' + msUs(est) + ' depois do ACK. Quem abriu o TCP não foi o seu navegador.' };
    }
    return { cls: 'warn', state: 'slow', title: 'Indício fraco', text: 'O TLS levou ' + msUs(gap) + ' a mais que o TCP, mas o ClientHello chegou logo após o ACK: mais provável Wi-Fi ou dispositivo lento que um proxy.' };
  }

  // utcMicros parses the RFC 3339 time of X-Traefix-Transport accepted= into microseconds of the day (UTC); the
  // Date type only keeps milliseconds.
  function utcMicros(s) {
    var m = /T(\d{2}):(\d{2}):(\d{2})(?:\.(\d+))?Z$/.exec(s || '');
    if (!m) return undefined;
    var frac = ((m[4] || '') + '000000').slice(0, 6);
    return ((Number(m[1]) * 60 + Number(m[2])) * 60 + Number(m[3])) * 1e6 + Number(frac);
  }

  function clockText(us) {
    var day = 86400e6;
    us = ((us % day) + day) % day;
    var p = function (n, w) { return String(n).padStart(w, '0'); };
    var sec = Math.floor(us / 1e6);
    return p(Math.floor(sec / 3600), 2) + ':' + p(Math.floor(sec / 60) % 60, 2) + ':' + p(sec % 60, 2) + '.' + p(us % 1e6, 6);
  }

  function msUs(us) {
    return (us < 0 ? '−' : '') + (Math.abs(us) / 1000).toFixed(Math.abs(us) < 10000 ? 2 : 1) + ' ms';
  }

  // timelineEvents rebuilds the milestones of the connection from X-Traefix-Transport (accepted) and
  // X-Traefix-Timing, in microseconds from the accept of the connection (the ACK of the handshake).
  function timelineEvents(timing, transport) {
    var ev = [];
    var at = function (us, name, what, seg) { if (us !== undefined && isFinite(us)) ev.push({ us: us, name: name, what: what, seg: seg }); };
    if (transport.proto === 'udp' || timing.proto === 'udp') {
      // Times from the accept of the connection, like over TCP: the Initial and the handshake count from the first
      // Initial, the request from the accept, which comes after the server processed that Initial (initial_to_acc_us).
      var acc = num(timing.initial_to_acc_us) || 0;
      var hs = num(timing.initial_to_hs_us), rq = num(timing.req_us);
      if (hs !== undefined) hs -= acc;
      at(-acc, 'Primeiro Initial do QUIC', 'Primeiro datagrama do cliente (ClientHello cifrado).');
      if (acc) at(0, 'Conexão aceita', 'O servidor processou o Initial e respondeu com o seu handshake.', 'srv');
      at(hs, 'Handshake QUIC concluído', 'TLS 1.3 dentro do QUIC, em 1 volta.', 'tls');
      // A request before the end of the handshake is 0-RTT data: on a resumed session the browser sends the request
      // in its first flight, without waiting for the server.
      var early = hs !== undefined && rq !== undefined && rq < hs;
      at(rq, 'Requisição HTTP', early ? 'Chegou antes do fim do handshake: 0-RTT, a requisição foi junto com o primeiro voo de uma sessão retomada.' : 'Esta requisição chegou ao Traefix.', 'req');
      ev.sort(function (a, c) { return a.us - c.us; });
      return ev;
    }
    var rtt = num(timing.tcp_rtt_us), est = num(timing.est_to_first_us);
    if (rtt !== undefined) at(-rtt, 'SYN (estimado)', 'Pedido de conexão. Estimado: aceite menos a volta TCP.');
    at(0, 'ACK: conexão aceita', 'Fim do handshake TCP com quem abriu a conexão.', 'tcp');
    if (est !== undefined) {
      at(est, 'Primeiro byte (ClientHello)', 'Direto, sai junto com o ACK (< 5 ms). Por proxy, só depois que o proxy avisa o cliente.', est > RELAY.first ? 'hot' : 'first');
      var ch = num(timing.ch_us), srv = num(timing.srv_us), tr = num(timing.tls_rtt_us), tlsUs = num(timing.tls_us);
      if (ch !== undefined) at(est + ch, 'ClientHello completo', 'Fim da mensagem inicial do TLS.', 'ch');
      if (ch !== undefined && srv !== undefined) {
        at(est + ch + srv, 'Resposta do servidor', 'ServerHello até o Finished do servidor, enviados.', 'srv');
        var gapUs = num(timing.rtt_gap_us);
        var slowTLS = gapUs !== undefined && rtt !== undefined && gapUs > RELAY.gap && gapUs > RELAY.frac * rtt;
        if (tr !== undefined) at(est + ch + srv + tr, 'Finished do cliente', 'Resposta do navegador: fecha a volta TLS, que vai até o navegador.', slowTLS ? (est > RELAY.first ? 'hot' : 'warm') : 'tls');
      }
      if (tlsUs !== undefined) at(est + tlsUs, 'Handshake TLS concluído', 'Conexão cifrada pronta.', 'hs');
    }
    at(num(timing.req_us), 'Requisição HTTP', 'Esta requisição chegou ao Traefix' + (num(transport.req_seq) > 1 ? ' (conexão reaproveitada: requisição ' + transport.req_seq + ')' : '') + '.', 'req');
    ev.sort(function (a, c) { return a.us - c.us; });
    return ev;
  }

  // timelineView draws the numbered gauge and the table of the milestones of one request: its connection's timing
  // (X-Traefix-Timing) and the accept time and position of the request (X-Traefix-Transport).
  function timelineView(timing, transport) {
    if (!timing || (!timing.proto && !timing.tcp_rtt_us)) return null;
    var ev = timelineEvents(timing, transport);
    if (!ev.length) return null;
    var base = utcMicros(transport.accepted);
    var t0 = ev[0].us;
    var tbl = el('table', { 'class': 'score-table timeline' });
    tbl.appendChild(el('thead', null, [el('tr', null, [
      el('th', { 'class': 'num', text: 'Nº' }), el('th', { text: 'Marco' }), el('th', { text: 'Horário (UTC)' }), el('th', { 'class': 'num', text: 'Desde o início' }), el('th', { 'class': 'num', text: 'Diferença' })
    ])]));
    var body = el('tbody');
    ev.forEach(function (e, i) {
      body.appendChild(el('tr', null, [
        el('td', { 'class': 'num' }, [el('span', { 'class': 'tl-num', text: String(i + 1) })]),
        el('td', null, [el('strong', { text: e.name }), el('div', { 'class': 'muted small', text: e.what })]),
        el('td', { 'class': 'mono', text: base !== undefined ? clockText(base + e.us) : '—' }),
        el('td', { 'class': 'num', text: msUs(e.us - t0) }),
        el('td', { 'class': 'num', text: i ? '+' + msUs(e.us - ev[i - 1].us) : '—' })
      ]));
    });
    tbl.appendChild(body);

    // Proportional bar of the intervals; the CSP forbids style attributes, the CSSOM is allowed.
    var total = ev[ev.length - 1].us - t0;
    var bar = el('div', { 'class': 'tl-bar' });
    if (total > 0) {
      for (var i = 1; i < ev.length; i++) {
        var w = (ev[i].us - ev[i - 1].us) / total * 100;
        if (w <= 0) continue;
        var seg = el('span', { 'class': 'tl-seg tl-' + (ev[i].seg || 'x'), title: ev[i - 1].name + ' → ' + ev[i].name + ': ' + msUs(ev[i].us - ev[i - 1].us) });
        seg.style.width = w + '%';
        bar.appendChild(seg);
      }
    }

    // Numbered gauge under the bar: the number of each mark is the one of its row in the table. Marks that would
    // overlap go to the next level down.
    var gauge = el('div', { 'class': 'tl-gauge' });
    var groups = [];
    ev.forEach(function (e, i) {
      var pos = total > 0 ? Math.min(100, Math.max(0, (e.us - t0) / total * 100)) : 0;
      var g = groups[groups.length - 1];
      if (g && pos - g.pos < 3.5) { g.items.push({ n: i + 1, e: e }); g.last = e; return; }
      groups.push({ pos: pos, first: e, last: e, items: [{ n: i + 1, e: e }] });
    });
    var lastAt = [];
    groups.forEach(function (g) {
      var level = 0;
      while (lastAt[level] !== undefined && g.pos - lastAt[level] < 9) level++;
      lastAt[level] = g.pos;
      var badges = el('span', { 'class': 'tl-badges' }, g.items.map(function (it) {
        return el('span', { 'class': 'tl-num', title: it.e.name + ': ' + msUs(it.e.us - t0), text: String(it.n) });
      }));
      var label = msUs(g.first.us - t0) + (g.last.us - g.first.us >= 10 ? ' a ' + msUs(g.last.us - t0) : '');
      // Near the ends the label grows inwards, so that it stays inside the card.
      var edge = g.pos > 85 ? ' tl-end' : (g.pos < 8 && g.items.length > 1 ? ' tl-start' : '');
      var mark = el('span', { 'class': 'tl-mark' + edge }, [
        el('span', { 'class': 'tl-line' }), badges, el('span', { 'class': 'tl-at', text: label })
      ]);
      mark.style.left = g.pos + '%';
      mark.style.top = (level * 40) + 'px';
      mark.firstChild.style.height = (level * 40 + 10) + 'px';
      mark.firstChild.style.top = (-level * 40 - 2) + 'px';
      gauge.appendChild(mark);
    });
    gauge.style.height = (lastAt.length * 40 + 6) + 'px';
    var scale = el('div', { 'class': 'tl-scale' }, [bar, gauge]);
    return [scale, el('div', { 'class': 'scroll' }, [tbl])];
  }

  function timelineCard(b) {
    var view = timelineView(timing, transport);
    if (!view) return null;
    var rl = relayVerdict();
    var rows = [];
    if (transport.proto !== 'udp' && timing.proto !== 'udp') {
      rows = [
        ['Volta TCP (servidor ↔ quem abriu o TCP)', timing.tcp_rtt_us !== undefined ? msUs(num(timing.tcp_rtt_us)) : undefined],
        ['Volta TLS (servidor ↔ navegador)', timing.tls_rtt_us !== undefined ? msUs(num(timing.tls_rtt_us)) : undefined],
        ['Diferença TLS − TCP', timing.rtt_gap_us !== undefined ? msUs(num(timing.rtt_gap_us)) + ' (limiar: > 10 ms e > 30% da volta TCP)' : undefined],
        ['ACK → primeiro byte', timing.est_to_first_us !== undefined ? msUs(num(timing.est_to_first_us)) + ' (limiar: > 5 ms)' : undefined]
      ];
    } else {
      rows = [['Volta QUIC (mínima)', timing.quic_min_rtt_us ? msUs(num(timing.quic_min_rtt_us)) : undefined]];
    }
    var n = b && b.navigation;
    if (n && n.connectMs > 0) {
      rows.push(['Visão do navegador', 'conexão ' + n.connectMs + ' ms' + (n.tlsMs ? ' (TLS ' + n.tlsMs + ' ms)' : '') + ', medido pelo próprio navegador: só para comparação, não pontua']);
    }
    rows.push(['Origem dos tempos', timing.src === 'kernel' ? 'kernel (horário de chegada dos pacotes)' : 'Traefix (horário de leitura)']);
    var verdict = rl ? el('div', { 'class': 'verdict ' + rl.cls }, [el('h3', { text: 'Diagnóstico: ' + rl.title }), el('p', { text: rl.text })]) : null;

    return card('Linha do tempo da conexão', 'Cada marco da abertura da sua conexão, como o servidor viu. O handshake TCP termina em quem abriu a conexão; a volta do TLS vai até o navegador. Com proxy ou SOCKS no caminho, os dois se separam e o primeiro byte atrasa: pela pontuação, isso pesa como automação (contorno de limites por IP).',
      view.concat([dl(rows), verdict]));
  }

  // ---------- requests of the session ----------

  var PROTOS = { h1: 'HTTP/1.1', 'http/1.1': 'HTTP/1.1', h2: 'HTTP/2', h2c: 'HTTP/2 (h2c)', h3: 'HTTP/3' };
  var DESTS = { document: 'página', script: 'script', style: 'estilo', image: 'imagem', font: 'fonte', empty: 'fetch', manifest: 'manifesto' };

  function protoText(p) { return PROTOS[p] || p || '?'; }

  function timeOfDay(rfc) {
    var us = utcMicros(rfc);
    return us === undefined ? '?' : clockText(us).slice(0, 12);
  }

  // sessionCard lists the requests the server saw with the session cookie, grouped by connection: a refresh that
  // switches from HTTP/2 to HTTP/3, a reused connection, the assets of the page. Each one opens its own timeline.
  function sessionCard() {
    var box = el('div', { 'class': 'session-list' }, [el('p', { 'class': 'muted', text: 'Carregando as requisições da sessão…' })]);
    var btn = el('button', { type: 'button', 'class': 'small-btn', text: 'Atualizar' });

    function draw(d) {
      box.textContent = '';
      var reqs = (d && d.requests) || [];
      if (!reqs.length) {
        box.appendChild(el('p', { 'class': 'muted', text: 'Nenhuma requisição registrada: o navegador recusou o cookie da sessão, ou a sessão expirou.' }));
        return;
      }
      var conns = [], byId = {};
      reqs.forEach(function (r) {
        var key = r.connId || ('sem-id-' + r.n);
        var c = byId[key];
        if (!c) { c = byId[key] = { id: r.connId, reqs: [], first: r }; conns.push(c); }
        c.reqs.push(r);
      });
      var mine = tx('Time');
      conns.forEach(function (c, ci) {
        var f = c.first;
        var transp = f.transport === 'udp' ? 'QUIC (UDP)' : (f.transport === 'tcp' ? 'TCP' : '');
        box.appendChild(el('h3', { 'class': 'session-conn' }, [
          'Conexão ' + (ci + 1) + ' · ' + protoText(f.proto) + (transp ? ' · ' + transp : '') + (f.accepted ? ' · aberta às ' + timeOfDay(f.accepted) : ''),
          c.id ? el('span', { 'class': 'muted mono small', text: ' ' + c.id }) : null
        ]));
        c.reqs.forEach(function (r) {
          var tm = parseKV(r.timing) || {};
          var seq = num(r.reqSeq);
          var reqUs = num(tm.req_us);
          var what = [
            '#' + r.n, timeOfDay(r.time), r.method + ' ' + r.path,
            DESTS[r.dest] || r.dest || '', String(r.status),
            seq ? (seq === 1 ? 'abriu a conexão' : 'requisição ' + seq + ' da conexão') : '',
            reqUs !== undefined ? msUs(reqUs) + ' após o aceite' : ''
          ].filter(Boolean).join(' · ');
          var det = el('details', { 'class': 'session-req' }, [el('summary', null, [
            what, r.time === mine ? el('span', { 'class': 'here', text: 'esta página' }) : null
          ])]);
          var built = false;
          det.addEventListener('toggle', function () {
            if (!det.open || built) return;
            built = true;
            var view = timelineView(tm, { proto: r.transport, accepted: r.accepted, req_seq: r.reqSeq });
            if (!view) { det.appendChild(el('p', { 'class': 'muted', text: 'Sem X-Traefix-Timing para esta requisição.' })); return; }
            if (seq > 1) det.appendChild(el('p', { 'class': 'muted small', text: 'Conexão reaproveitada: os marcos do handshake são os da abertura da conexão; o último marco é esta requisição.' }));
            view.forEach(function (v) { det.appendChild(v); });
          });
          if (r.time === mine) det.open = true;
          box.appendChild(det);
        });
      });
      if (reqs.length >= d.max) box.appendChild(el('p', { 'class': 'muted small', text: 'Mostrando as ' + d.max + ' requisições mais recentes.' }));
    }

    function load() {
      fetch('/api/session', { cache: 'no-store', credentials: 'same-origin' })
        .then(function (r) { return r.json(); })
        .then(draw)
        .catch(function () { box.textContent = ''; box.appendChild(el('p', { 'class': 'muted', text: 'Não foi possível ler as requisições da sessão.' })); });
    }
    btn.addEventListener('click', load);
    // After the assets of the page, so that they are in the list too.
    if (document.readyState === 'complete') setTimeout(load, 300);
    else window.addEventListener('load', function () { setTimeout(load, 300); });

    return card('Requisições desta sessão', 'Cada requisição que o servidor recebeu deste navegador nos últimos 30 minutos, agrupada por conexão: o recarregamento que troca HTTP/2 por HTTP/3, a conexão reaproveitada, os arquivos da página. Abra uma para ver a linha do tempo dela. Um cookie de sessão (infopage_sid, 30 min) agrupa as requisições; elas ficam só na memória do servidor.',
      [el('div', { 'class': 'session-actions' }, [btn]), box]);
  }

  // evaluate grades the IP and TCP values per infopage/CRITERIA.md: most of them only describe the
  // operating system or the access network, and stay informative.
  function evaluate() {
    var r = [];
    var add = function (cls, name, value, text) { r.push([cls, name, value, text]); };
    var v6 = transport.ipv === '6';

    var rtt = pathRTT();
    if (rtt !== undefined) {
      var m = rtt / 1000;
      if (m < 30) add('ok', 'Latência', ms(rtt), 'Excelente para jogos, chamadas e navegação.');
      else if (m < 80) add('ok', 'Latência', ms(rtt), 'Boa para qualquer uso.');
      else if (m < 300) add('info', 'Latência', ms(rtt), 'Depende da distância até o servidor (no Brasil): normal se você estiver longe; perto, pode indicar Wi-Fi ou rede congestionada.');
      else add('warn', 'Latência', ms(rtt), 'Alta: chamadas e jogos sofrem. Comum em satélite geoestacionário.');
    }

    var mt = mtuEstimate();
    if (mt && mt.mtu !== undefined) {
      add(mtuClass(mt.grade), 'MTU estimado', mt.mtu + ' bytes · nota ' + mt.grade + '/100', mt.base + ' ' + mt.payload + ' + ' + mt.l4Name + ' ' + mt.l4 + ' + IPv' + mt.ipv + ' ' + mt.ip + '. ' + mtuReading(mt.mtu) + ' Régua no cartão "MTU estimado".');
    } else if (mt) {
      add('info', 'MTU estimado', 'sem plano', mt.note);
    }

    var ws = num(transport.syn_wscale);
    if (transport.proto === 'tcp') {
      if (ws === undefined || ws === 0) add('warn', 'Escala de janela', ws === 0 ? '0' : 'ausente', 'Sem escala, a janela fica em 64 KB: downloads longos ficam limitados (cerca de 10 Mbps a 50 ms).');
      else add(ws >= 5 ? 'ok' : 'info', 'Escala de janela', '2^' + ws, 'Permite janelas de até ' + Math.round(65535 * Math.pow(2, ws) / 1048576 * 10) / 10 + ' MB; o sistema ajusta a janela real conforme a necessidade.');
    }

    if (transport.syn_sack !== undefined) {
      if (transport.syn_sack === '1') add('ok', 'SACK', 'ativo', 'Recupera pacotes perdidos sem reenviar tudo.');
      else add('warn', 'SACK', 'ausente', 'Perdas custam mais: pilha antiga ou equipamento no caminho removendo a opção.');
    }
    if (transport.syn_ts !== undefined) {
      add('info', 'Timestamps TCP', transport.syn_ts === '1' ? 'ativos' : 'desativados', transport.syn_ts === '1' ? 'Padrão de Linux, Android, macOS e iOS.' : 'Padrão do Windows; não é problema.');
    }
    if (transport.syn_ecn !== undefined) {
      add('info', 'ECN', transport.syn_ecn === '1' ? 'pedido' : 'não pedido', transport.syn_ecn === '1' ? 'Seu sistema pede aviso de congestionamento sem perda (Apple faz isso).' : 'Windows e Linux não pedem por padrão; não é problema.');
    }
    if (transport.syn_tfo === '1') add('info', 'TCP Fast Open', 'usado', 'Atalho de abertura de conexão, raro na web.');

    if (transport.ttl_init !== undefined) {
      add('info', 'Saltos de rede', (transport.hops || '?') + ' (TTL ' + (transport.ttl || '?') + ')', 'Quantos roteadores o pacote atravessou; mais saltos não significa conexão pior.');
    }

    if (!v6 && transport.df === '0') add('warn', 'Não fragmentar (DF)', 'desligado', 'Incomum: algum equipamento ou VPN apagou a marca no caminho.');
    if (num(transport.ipopt) > 0) add('warn', 'Opções IP', transport.ipopt + ' bytes', 'Navegadores não usam opções IP: pacote atípico.');
    if (transport.ecn === '3') add('warn', 'ECN no IP', 'CE', 'Um equipamento do caminho altera os bits de congestionamento.');
    if (num(transport.dscp) > 0) add('info', 'Prioridade (DSCP)', transport.dscp, 'Marcada pela sua rede ou remarcada pelo provedor; sem efeito na internet.');

    var re = num(transport.retrans);
    if (re !== undefined) {
      if (re === 0) add('info', 'Reenvios', '0', 'Nenhum reenvio nesta conexão. Uma conexão curta não mede perda de pacotes.');
      else add('info', 'Reenvios', String(re), 'Houve reenvio nesta conexão; uma ocorrência isolada não indica problema.');
    }
    return r;
  }

  function signals(b, os) {
    var s = [];
    var add = function (cls, text) { s.push([cls, text]); };

    if (b.navigator.webdriver) add('bad', 'O navegador declara estar sob controle de automação (navigator.webdriver).');

    if (tls.ver === '1.3') add('ok', 'Conexão cifrada com TLS 1.3, a versão mais recente.');
    else if (tls.ver) add('warn', 'Conexão cifrada com TLS ' + tls.ver + '; o TLS 1.3 é mais seguro e rápido.');
    else if (location.protocol !== 'https:') add('bad', 'Conexão sem criptografia (HTTP).');

    if (/MLKEM|Kyber/i.test(tls.group || '')) add('ok', 'Troca de chaves pós-quântica (' + tls.group + ').');
    if (tls.resumed === '1') add('ok', 'Sessão TLS retomada: a conexão foi reaproveitada de uma visita anterior.');
    if (tls.ech === '1') add('ok', 'Encrypted Client Hello aceito: o nome do site viajou cifrado.');

    var fam = osFamily(os);
    var ttlFam = transport.ttl_init === '128' ? 'windows' : transport.ttl_init === '64' ? 'unix' : '';
    if (fam && ttlFam) {
      if (fam === ttlFam) add('ok', 'O sistema sugerido pela rede (TTL ' + transport.ttl_init + ') combina com o informado pelo navegador.');
      else add('warn', 'O sistema sugerido pela rede (TTL ' + transport.ttl_init + ': ' + ttlOS(transport.ttl_init) + ') difere do informado pelo navegador (' + os + '): comum com VPN, proxy ou roteador.');
    }

    var rl = relayVerdict();
    if (rl) add(rl.cls, rl.text);

    var ua = b.navigator.userAgent || '';
    if (/Chrome\//.test(ua) && !/Edg|OPR|CriOS|SamsungBrowser/.test(ua) && b.features.secureContext && !b.userAgentData) {
      add('warn', 'O navegador se diz Chromium mas não expõe Client Hints (navigator.userAgentData).');
    }

    if (b.navigator.globalPrivacyControl || b.navigator.doNotTrack === '1') add('ok', 'Você pede aos sites para não ser rastreado (GPC/DNT).');
    if (!b.navigator.cookieEnabled) add('warn', 'Cookies desativados: muitos sites não funcionarão.');

    if (b.clock.skewMs !== undefined && Math.abs(b.clock.skewMs) > 60000) {
      var skew = Math.abs(b.clock.skewMs) > 300000 ? 'bad' : 'warn';
      add(skew, 'O relógio do seu dispositivo difere do servidor em ' + Math.round(b.clock.skewMs / 1000) + ' s' + (skew === 'bad' ? ': logins com código temporário e certificados podem falhar.' : '.'));
    }
    return s;
  }


  // ---------- automation score (infopage/SCORE.md) ----------

  // The server computes the passive rules from the request (data.score); the page adds the rules of the browser
  // APIs, family js, which the client controls, and aggregates with the same caps and calibration.
  var STATE = {
    ideal: ['ok', 'Ideal'], normal: ['info', 'Normal'], atipico: ['amber', 'Atípico'],
    contradicao: ['orange', 'Contradição'], declarado: ['bad', 'Declarado'], desconhecido: ['muted', 'Sem dado']
  };
  var ROLES = {
    navegador: 'Navegador', automacao_declarada: 'Automação declarada', crawler_verificado: 'Robô verificado',
    crawler_nao_verificado: 'Robô não verificado', via_proxy: 'Navegador por proxy ou SOCKS', desconhecido: 'Cliente não identificado'
  };
  var FAMILIES = { decl: 'Declaração', tls: 'TLS', http: 'HTTP', tcp: 'TCP/IP', time: 'Tempo', net: 'Rede', js: 'Navegador (JS)' };

  function firstLang(s) {
    return String(s || '').split(',')[0].split(';')[0].trim().toLowerCase().split('-')[0];
  }

  function jsRules(b, sc) {
    var r = [];
    var add = function (o) { o.family = o.family || 'js'; o.points = o.points || 0; r.push(o); };
    var c = sc.client || {};
    var nav = b.navigator;

    if (nav.webdriver) {
      add({ id: 'DECL-WD', family: 'decl', name: 'navigator.webdriver', value: 'true', expected: 'false', state: 'declarado', points: 50, weight: 50,
        reason: 'O navegador declara estar sob controle de automação.', benign: 'Testes automatizados do próprio dono do site.' });
    }
    if (!c.engine || c.crawler) return r;

    var headerUA = hdr('User-Agent') || '';
    var o = { id: 'JS-UA', name: 'User-Agent do JS × cabeçalho', value: nav.userAgent === headerUA ? 'iguais' : 'diferentes', expected: 'iguais', weight: 30, state: 'ideal' };
    if (nav.userAgent !== headerUA) { o.state = 'contradicao'; o.points = 30; o.reason = 'navigator.userAgent difere do User-Agent enviado.'; o.benign = 'Extensão que troca o UA só no cabeçalho.'; }
    add(o);

    var chPlat = (hdr('Sec-Ch-Ua-Platform') || '').replace(/"/g, '');
    if (b.userAgentData && chPlat) {
      o = { id: 'JS-CH-HDR', name: 'Client Hints do JS × cabeçalho', value: b.userAgentData.platform, expected: chPlat, weight: 30, state: 'ideal' };
      var mob = hdr('Sec-Ch-Ua-Mobile');
      if (b.userAgentData.platform !== chPlat || (mob && (mob === '?1') !== !!b.userAgentData.mobile)) {
        o.state = 'contradicao'; o.points = 30; o.reason = 'Os Client Hints do cabeçalho não são os da API: cabeçalho fabricado por ferramenta.';
      }
      add(o);
    }

    var plat = nav.platform || '';
    var want = { windows: /^Win/, macos: /^Mac/, ios: /^(iPhone|iPad|iPod|Mac)/, android: /Linux|Android|arm/i, linux: /Linux/, chromeos: /Linux|CrOS/ }[c.os];
    if (want && plat) {
      o = { id: 'JS-PLAT', name: 'navigator.platform', value: plat, expected: c.os, weight: 20, state: 'ideal' };
      if (!want.test(plat)) { o.state = 'contradicao'; o.points = 20; o.reason = 'A plataforma do JavaScript não é a do User-Agent.'; }
      add(o);
    }

    if (b.webgl && b.webgl.renderer) {
      o = { id: 'JS-GPU', name: 'Placa de vídeo (WebGL)', value: b.webgl.renderer, expected: 'GPU real', weight: 15, state: 'ideal' };
      if (/SwiftShader|llvmpipe|softpipe|Basic Render/i.test(b.webgl.renderer)) {
        o.state = 'atipico'; o.points = 15; o.reason = 'Renderização por software, padrão do Chrome headless.'; o.benign = 'Máquina virtual, RDP, GPU desativada.';
      }
      add(o);
    } else {
      add({ id: 'JS-GPU', name: 'Placa de vídeo (WebGL)', state: 'desconhecido', weight: 15, reason: 'WebGL indisponível.' });
    }

    var al = hdr('Accept-Language');
    if (al && nav.languages) {
      o = { id: 'JS-LANG', name: 'Idioma do JS × Accept-Language', value: nav.language, expected: al.split(',')[0], weight: 10, state: 'ideal' };
      if (firstLang(nav.languages) !== firstLang(al)) { o.state = 'contradicao'; o.points = 10; o.reason = 'O idioma do navegador não é o do cabeçalho.'; }
      add(o);
    }

    if (c.mobile) {
      o = { id: 'JS-TOUCH', name: 'Toque em celular', value: String(nav.maxTouchPoints), expected: 'mais de 0', weight: 15, state: 'ideal' };
      if (!nav.maxTouchPoints) { o.state = 'contradicao'; o.points = 15; o.reason = 'UA de celular sem tela de toque.'; o.benign = 'Emulação de dispositivo do DevTools.'; }
      add(o);
    }

    var sc2 = b.screen;
    // outerWidth is 0 in embedded webviews too: only the screen counts.
    o = { id: 'JS-SCREEN', name: 'Tela', value: sc2.width + '×' + sc2.height, expected: 'dimensões reais', weight: 20, state: 'ideal' };
    if (!sc2.width || !sc2.height) { o.state = 'contradicao'; o.points = 20; o.reason = 'Tela de tamanho zero, típico de navegador sem interface.'; }
    add(o);

    if (c.engine === 'chromium' && !c.mobile && nav.pdfViewerEnabled === false) {
      add({ id: 'JS-PDF', name: 'Leitor de PDF', value: 'não', expected: 'sim', state: 'atipico', points: 5, weight: 5,
        reason: 'Chrome desktop tem leitor de PDF embutido.', benign: 'Política corporativa.' });
    }

    var n = 0, pos = false;
    r.forEach(function (x) { if (x.family === 'js' && x.state !== 'desconhecido') { n++; if (x.points > 0) pos = true; } });
    if (n >= 3 && !pos) add({ id: 'JS-MATCH', name: 'APIs coerentes', state: 'ideal', points: -10, weight: 10, reason: 'APIs do navegador coerentes com o User-Agent.' });
    return r;
  }

  function aggregateScore(results, sc) {
    var sum = {}, evaluated = 0, total = 0;
    results.forEach(function (x) {
      sum[x.family] = (sum[x.family] || 0) + x.points;
      var w = Math.abs(x.weight || 0);
      total += w;
      if (x.state !== 'desconhecido') evaluated += w;
    });
    var fams = {}, S = 0;
    Object.keys(sum).forEach(function (f) {
      var cap = (sc.families || {})[f] || { max: 0, min: 0 };
      var v = Math.min(Math.max(sum[f], cap.min || 0), cap.max || 0);
      fams[f] = v;
      S += v;
    });
    var R = Math.round(100 / (1 + Math.exp(-(sc.a * S + sc.b))));
    var cov = total ? Math.round(100 * evaluated / total) : 0;
    var band = cov < 50 ? 'Evidência insuficiente' : R <= 20 ? 'Compatível com navegador' : R <= 50 ? 'Indeterminado' : R <= 80 ? 'Suspeito de automação' : 'Automação provável';
    return { S: S, R: R, coverage: cov, band: band, families: fams, raw: sum, evaluated: evaluated, weight: total };
  }

  function fullScore(b) {
    var sc = data.score;
    if (!sc) return null;
    var results = sc.results.concat(jsRules(b, sc));
    var role = sc.role;
    if (b.navigator.webdriver && role === 'navegador') role = 'automacao_declarada';
    if (role === 'automacao_declarada' || role === 'via_proxy') {
      var why = role === 'via_proxy' ? ' Sem alívio: proxy ou SOCKS no caminho.' : ' Sem alívio: automação declarada.';
      results = results.map(function (x) {
        return x.points < 0 ? Object.assign({}, x, { points: 0, reason: (x.reason || '') + why }) : x;
      });
    }
    return { sc: sc, role: role, results: results, total: aggregateScore(results, sc) };
  }

  function bandClass(t) {
    if (t.coverage < 50) return 'muted';
    return t.R <= 20 ? 'ok' : t.R <= 50 ? 'info' : t.R <= 80 ? 'warn' : 'bad';
  }

  function scoreCard(f) {
    var t = f.total, sc = f.sc, net = sc.network || {};
    var head = el('div', { 'class': 'score-head' }, [
      el('div', { 'class': 'score-value ' + bandClass(t), text: String(t.R) }),
      el('div', null, [
        el('div', { 'class': 'score-band ' + bandClass(t), text: t.band }),
        el('div', { 'class': 'muted', text: ROLES[f.role] + ' · cobertura ' + t.coverage + '% · índice não calibrado, de 0 a 100' })
      ])
    ]);

    var facts = [];
    if (sc.client && sc.client.known) facts.push(['TLS reconhecido', sc.client.known + ' (medido em laboratório)']);
    if (net.relay) facts.push(['Relay de privacidade', net.relay + (net.relayDetail ? ' · ' + net.relayDetail.replace(/,/g, ', ') : '')]);
    if (net.cloud) facts.push(['Rede de datacenter', net.cloud]);
    if (net.crawlers && net.crawlers.length) facts.push(['Endereço de robô', net.crawlerLabel + ' (' + net.crawlers.join(', ') + ')']);

    return card('Pontuação de automação', 'Quanto o seu acesso se parece com um programa, pela coerência entre o que o navegador declara e o que a conexão mostra. Todos os itens analisados aparecem abaixo, agrupados por tipo de análise, com o resultado, os pontos e a soma de cada tipo. Valores fora do ideal de rede (MSS, latência) não contam.',
      [head, facts.length ? dl(facts) : null, scoreBreakdown(f), scoreVerdict(f)]);
  }

  function dec(n) {
    return String(n).replace('.', ',');
  }

  function signed(n) {
    return (n > 0 ? '+' : n < 0 ? '−' : '') + Math.abs(n);
  }

  // relayTimeline shows, under the TIME-RELAY rule, how long each part of the opening of the connection took. The
  // parts that give a proxy or SOCKS away are in red: the wait between the ACK and the ClientHello, and what the TLS
  // round trip took beyond the TCP one. With only one of the two signals, the part is in orange (weak hint).
  function relayTimeline() {
    if (transport.proto === 'udp' || timing.proto === 'udp') return null;
    var rtt = num(timing.tcp_rtt_us), est = num(timing.est_to_first_us), ch = num(timing.ch_us), srv = num(timing.srv_us);
    var tr = num(timing.tls_rtt_us), gap = num(timing.rtt_gap_us), tlsUs = num(timing.tls_us), rq = num(timing.req_us);
    if (rtt === undefined || est === undefined || tr === undefined || gap === undefined) return null;
    var slow = gap > RELAY.gap && gap > RELAY.frac * rtt;
    var late = est > RELAY.first;
    var hot = slow && late ? 'bad' : 'orange';
    var parts = [{ name: 'Handshake TCP (SYN → ACK)', us: rtt, cls: 'tcp' }];
    parts.push({
      name: 'ACK → ClientHello', us: Math.max(est, 0), cls: late ? hot : 'n1', mark: late,
      note: late ? 'acima de ' + msUs(RELAY.first) + ': o cliente esperou o aviso do proxy' : 'direto: sai junto com o ACK'
    });
    if (ch !== undefined && srv !== undefined) parts.push({ name: 'ClientHello → resposta do servidor', us: ch + srv, cls: 'n2' });
    var base = slow ? Math.max(tr - gap, 0) : tr;
    parts.push({ name: 'Volta TLS' + (slow ? ' (parte igual à volta TCP)' : ' (resposta → Finished do cliente)'), us: base, cls: 'tls' });
    if (slow) parts.push({ name: 'Volta TLS além da volta TCP', us: gap, cls: hot, mark: true, note: 'trecho entre o proxy e o navegador' });
    if (tlsUs !== undefined && rq !== undefined && rq > est + tlsUs) parts.push({ name: 'Fim do handshake → requisição', us: rq - est - tlsUs, cls: 'n1' });

    var total = parts.reduce(function (a, p) { return a + p.us; }, 0);
    if (!(total > 0)) return null;
    var bar = el('div', { 'class': 'rt-bar' });
    var list = el('ul', { 'class': 'rt-parts' });
    parts.forEach(function (p) {
      if (p.us > 0) {
        var seg = el('span', { 'class': 'rt-seg rt-' + p.cls, title: p.name + ': ' + msUs(p.us) });
        seg.style.width = (p.us / total * 100) + '%';
        bar.appendChild(seg);
      }
      var tone = p.mark ? (p.cls === 'bad' ? 'hot' : 'warm') : '';
      list.appendChild(el('li', null, [
        el('span', { 'class': 'rt-dot rt-' + p.cls }),
        el('span', { 'class': tone }, [p.name + ': ', el('strong', { text: msUs(p.us) }), p.note ? ' · ' + p.note : ''])
      ]));
    });
    return el('div', null, [bar, list]);
  }

  // scoreBreakdown is the table of every rule evaluated, grouped by family, with the raw sum of each family,
  // the sum after the family cap, and the total S and index R.
  function scoreBreakdown(f) {
    var t = f.total, sc = f.sc;
    var tbl = el('table', { 'class': 'score-table' });
    tbl.appendChild(el('thead', null, [el('tr', null, [
      el('th', { text: 'Item analisado' }),
      el('th', { text: 'Observado / esperado' }),
      el('th', { 'class': 'res', text: 'Resultado' }),
      el('th', { 'class': 'num', text: 'Pontos' })
    ])]));

    var order = Object.keys(FAMILIES);
    f.results.forEach(function (x) { if (order.indexOf(x.family) < 0) order.push(x.family); });
    var body = el('tbody');
    order.forEach(function (k) {
      var rows = f.results.filter(function (x) { return x.family === k; });
      if (!rows.length) return;
      var cap = sc.families[k] || { max: 0, min: 0 };
      body.appendChild(el('tr', { 'class': 'fam' }, [el('th', { colspan: '4', text: (FAMILIES[k] || k) + ' · teto ' + signed(cap.max) + (cap.min ? ' / ' + signed(cap.min) : '') })]));
      rows.forEach(function (x) {
        var st = STATE[x.state] || STATE.normal;
        body.appendChild(el('tr', null, [
          el('td', null, [el('strong', { text: x.name }), el('span', { 'class': 'muted', text: ' ' + x.id }),
            x.reason ? el('div', { 'class': 'muted small', text: x.reason + (x.benign ? ' Explicação benigna: ' + x.benign : '') }) : null]),
          el('td', null, [x.value ? el('div', { text: x.value }) : el('div', { 'class': 'muted', text: '—' }),
            x.expected ? el('div', { 'class': 'muted small', text: 'esperado: ' + x.expected }) : null]),
          el('td', { 'class': 'res' }, [badge(st[1], st[0])]),
          el('td', { 'class': 'num ' + (x.points > 0 ? 'bad' : x.points < 0 ? 'ok' : 'muted'), text: signed(x.points) })
        ]));
        if (x.id === 'TIME-RELAY') {
          var rt = relayTimeline();
          if (rt) body.appendChild(el('tr', { 'class': 'rule-extra' }, [el('td', { colspan: '4' }, [rt])]));
        }
      });
      var raw = t.raw[k] || 0, v = t.families[k] || 0;
      var capped = raw !== v ? ' → ' + signed(v) + ' (limitado pelo teto)' : '';
      body.appendChild(el('tr', { 'class': 'subtotal' }, [
        el('td', { colspan: '3', text: 'Soma ' + (FAMILIES[k] || k) + ': ' + rows.length + (rows.length === 1 ? ' item' : ' itens') + ', ' + signed(raw) + capped }),
        el('td', { 'class': 'num', text: signed(v) })
      ]));
    });
    tbl.appendChild(body);

    var parts = order.filter(function (k) { return k in t.families; }).map(function (k) { return (FAMILIES[k] || k) + ' ' + signed(t.families[k]); });
    tbl.appendChild(el('tfoot', null, [
      el('tr', null, [el('td', { colspan: '3', text: 'Total S = ' + parts.join(' ') }), el('td', { 'class': 'num', text: signed(t.S) })]),
      el('tr', null, [el('td', { colspan: '3', text: 'Índice R = 100 / (1 + e^−(' + dec(sc.a) + ' × S ' + (sc.b < 0 ? '− ' + dec(Math.abs(sc.b)) : '+ ' + dec(sc.b)) + '))' }), el('td', { 'class': 'num', text: t.R + '/100' })]),
      el('tr', null, [el('td', { colspan: '3', text: 'Cobertura: peso dos itens com dado sobre o peso de todos (' + t.evaluated + ' de ' + t.weight + ')' }), el('td', { 'class': 'num', text: t.coverage + '%' })])
    ]));
    return el('div', { 'class': 'scroll' }, [tbl]);
  }

  // scoreVerdict is the full evaluation in words: the band, the role, and what weighed each way.
  function scoreVerdict(f) {
    var t = f.total;
    var byPoints = function (a, c) { return Math.abs(c.points) - Math.abs(a.points); };
    var pos = f.results.filter(function (x) { return x.points > 0; }).sort(byPoints);
    var neg = f.results.filter(function (x) { return x.points < 0; }).sort(byPoints);
    var unknown = f.results.filter(function (x) { return x.state === 'desconhecido'; });
    var fmt = function (l) { return l.map(function (x) { return x.name + ' (' + signed(x.points) + ')'; }).join('; '); };
    var capped = Object.keys(t.families).filter(function (k) { return (t.raw[k] || 0) !== t.families[k]; })
      .map(function (k) { return FAMILIES[k] || k; });

    var lines = [
      ['Resultado', t.band + ' · índice ' + t.R + '/100 · S = ' + signed(t.S)],
      ['Papel do cliente', ROLES[f.role] || f.role],
      ['Itens analisados', f.results.length + ' (' + (f.results.length - unknown.length) + ' com dado, ' + unknown.length + ' sem dado)'],
      ['A favor de automação', pos.length ? fmt(pos) : 'nenhum item'],
      ['A favor de navegador', neg.length ? fmt(neg) : 'nenhum item'],
      ['Tetos aplicados', capped.length ? capped.join(', ') + ': itens da mesma origem contam como uma evidência' : undefined],
      ['Sem dado', unknown.length ? unknown.map(function (x) { return x.name; }).join('; ') : undefined]
    ];
    var note = t.coverage < 50
      ? 'Menos da metade do peso das regras tem dado: a faixa não é conclusiva.'
      : 'Índice heurístico, não calibrado: descreve a coerência técnica do acesso, não prova que é uma pessoa ou um programa.';
    return el('div', { 'class': 'verdict ' + bandClass(t) }, [el('h3', { text: 'Avaliação completa' }), dl(lines), el('p', { 'class': 'muted small', text: note })]);
  }

  function scoreTable(f) {
    var t = el('table', { 'class': 'eval' });
    f.results.forEach(function (x) {
      var st = STATE[x.state] || STATE.normal;
      t.appendChild(el('tr', null, [
        el('td', { 'class': 'eval-badge' }, [badge(st[1], st[0])]),
        el('th', { text: x.name + ' (' + x.id + ')' }),
        el('td', null, [
          el('strong', { text: (x.points > 0 ? '+' : '') + x.points + ' ' }),
          el('span', { text: FAMILIES[x.family] + (x.value ? ' · ' + x.value : '') + (x.expected ? ' · esperado: ' + x.expected : '') }),
          x.reason ? el('div', { 'class': 'muted', text: x.reason + (x.benign ? ' Explicação benigna: ' + x.benign : '') }) : null
        ])
      ]));
    });
    return card('Regras da pontuação (' + f.sc.version + ')', 'Cada regra com o valor observado, o esperado para o cliente declarado, o estado e os pontos. Família com teto: TLS ' + f.sc.families.tls.max + ', HTTP ' + f.sc.families.http.max + ', TCP/IP ' + f.sc.families.tcp.max + '. S = ' + f.total.S + '.', [t]);
  }

  // ---------- render: Resumo ----------

  function renderResumo(b) {
    var root = document.getElementById('resumo');
    var ip = tx('Src-Ip') || hdr('X-Real-Ip') || (hdr('X-Forwarded-For') || '').split(',')[0].trim() || hostOf(data.remoteAddr);
    var os = osName(b);
    var rtt = ms(pathRTT());

    root.appendChild(el('div', { 'class': 'card hero' }, [
      el('div', null, [el('div', { 'class': 'label', text: 'Seu IP' }), el('div', { 'class': 'value', text: ip || '?' })]),
      el('div', null, [el('div', { 'class': 'label', text: 'Navegador' }), el('div', { 'class': 'value', text: browserName(b) })]),
      el('div', null, [el('div', { 'class': 'label', text: 'Sistema' }), el('div', { 'class': 'value', text: os })])
    ]));

    var fs = fullScore(b);
    if (fs) root.appendChild(scoreCard(fs));

    var ipv = transport.ipv ? 'IPv' + transport.ipv : (ip && ip.indexOf(':') >= 0 ? 'IPv6' : 'IPv4');
    root.appendChild(card('Sua conexão', 'Como o seu acesso chegou até este servidor.', [dl([
      ['Endereço', ip ? ip + ' (' + ipv + ')' : undefined],
      ['Porta de origem', tx('Src-Port')],
      ['Protocolo', protoName()],
      ['Segurança', tls.ver ? 'TLS ' + tls.ver + ' · ' + (tls.cipher || '') : (location.protocol === 'https:' ? 'HTTPS' : 'Sem criptografia')],
      ['Troca de chaves', tls.group],
      ['Nome do site (SNI)', tls.sni ? pct(tls.sni) : undefined],
      ['Latência (ida e volta)', rtt],
      ['Distância estimada', transport.hops ? transport.hops + ' saltos de rede' : undefined],
      ['Sistema pela rede', transport.ttl_init ? ttlOS(transport.ttl_init) + ' (TTL inicial ' + transport.ttl_init + ')' : undefined],
      ['Tipo de rede (navegador)', b.network ? [b.network.type, b.network.effectiveType].filter(Boolean).join(' · ') : undefined],
      ['Identificador da conexão', tx('Conn-Id')]
    ])]));

    var tc = timelineCard(b);
    if (tc) root.appendChild(tc);
    root.appendChild(sessionCard());

    var mc = mtuCard();
    if (mc) root.appendChild(mc);

    var sc = b.screen;
    root.appendChild(card('Seu navegador e dispositivo', 'O que o navegador conta a qualquer site que você visita.', [dl([
      ['Navegador', browserName(b)],
      ['Sistema operacional', os],
      ['Dispositivo', deviceKind(b)],
      ['Arquitetura', b.userAgentData && b.userAgentData.architecture ? b.userAgentData.architecture + (b.userAgentData.bitness ? ' ' + b.userAgentData.bitness + ' bits' : '') : undefined],
      ['Idiomas', b.navigator.languages],
      ['Fuso horário', b.intl.timeZone + ' (UTC' + (b.intl.utcOffsetMinutes >= 0 ? '+' : '-') + Math.abs(b.intl.utcOffsetMinutes / 60) + ')'],
      ['Tela', sc.width + ' × ' + sc.height + ' px · escala ' + sc.devicePixelRatio + 'x · ' + sc.colorDepth + ' bits de cor'],
      ['Janela', sc.viewport],
      ['Tema preferido', b.media.colorScheme === 'dark' ? 'Escuro' : 'Claro'],
      ['Toque', b.navigator.maxTouchPoints > 0 ? 'Sim (' + b.navigator.maxTouchPoints + ' pontos)' : 'Não'],
      ['Processadores', b.navigator.hardwareConcurrency ? b.navigator.hardwareConcurrency + ' núcleos' : undefined],
      ['Memória', b.navigator.deviceMemory ? '~' + b.navigator.deviceMemory + ' GB' : undefined],
      ['Placa de vídeo', b.webgl ? b.webgl.renderer : undefined],
      ['Cookies', b.navigator.cookieEnabled ? 'Ativados' : 'Desativados']
    ])]));

    var ev = evaluate();
    if (ev.length) {
      var t = el('table', { 'class': 'eval' });
      ev.forEach(function (r) {
        t.appendChild(el('tr', null, [
          el('td', { 'class': 'eval-badge' }, [badge(LABELS[r[0]], r[0])]),
          el('th', { text: r[1] }),
          el('td', null, [el('strong', { text: r[2] }), el('span', { 'class': 'muted', text: ' ' + r[3] })])
        ]));
      });
      root.appendChild(card('Avaliação TCP/IP', 'O que os pacotes da sua conexão mostram. A maior parte só descreve o seu sistema e a sua rede (azul): não existe valor "melhor".', [t]));
    }

    var sig = signals(b, os);
    if (sig.length) {
      var ul = el('ul', { 'class': 'signals' });
      sig.forEach(function (x) {
        ul.appendChild(el('li', null, [badge(LABELS[x[0]], x[0]), ' ', x[1]]));
      });
      root.appendChild(card('Sinais', 'Coerência entre o que o navegador diz e o que a conexão mostra.', [ul]));
    }

    var fps = [
      ['TLS (JA4)', tx('Ja4')],
      ['TLS (JA3)', tx('Ja3')],
      ['Requisição HTTP (JA4H)', tx('Ja4h')],
      ['HTTP/2 (Akamai)', tx('H2')],
      ['TCP (JA4T)', tx('Ja4t')]
    ].filter(function (r) { return r[1]; });
    if (fps.length) {
      root.appendChild(card('Impressões digitais', 'Resumos da forma como seu navegador abre conexões. Identificam o tipo de cliente (navegador, versão, sistema), não você: são iguais para quem usa o mesmo software.', [dl(fps)]));
    } else {
      root.appendChild(card('Impressões digitais', null, [el('p', { 'class': 'muted', text: 'O proxy não enviou cabeçalhos ' + data.prefix + '*: a opção fingerprint não está ativa no entrypoint, ou esta página não está atrás do Traefix.' })]));
    }
  }

  // ---------- render: Técnico ----------

  function renderTecnico(b) {
    var root = document.getElementById('tecnico');

    root.appendChild(card('Requisição', null, [kvTable({
      'Método': data.method,
      'URL': data.url,
      'Host': data.host,
      'Protocolo (no backend)': data.proto,
      'Protocolo (no proxy)': tx('Proto'),
      'Protocolo (navegador)': b.navigation ? b.navigation.nextHopProtocol : undefined,
      'RemoteAddr': data.remoteAddr
    })]));

    // Headers in the order the client sent them, when Traefix gives it.
    var own = [], other = [];
    Object.keys(headers).forEach(function (k) {
      (k.indexOf(prefix) === 0 ? own : other).push(k);
    });
    own.sort();
    var order = (tx('Header-Order') || '').toLowerCase().split(',');
    other.sort(function (a, c) {
      var ia = order.indexOf(a), ic = order.indexOf(c);
      if (ia < 0) ia = 1e6;
      if (ic < 0) ic = 1e6;
      return ia - ic || (a < c ? -1 : 1);
    });

    var t1 = el('table');
    own.forEach(function (k) {
      t1.appendChild(el('tr', null, [el('th', { text: headers[k].name }), el('td', null, [valueCell(headers[k].value)])]));
    });
    root.appendChild(card('Cabeçalhos do Traefix (' + data.prefix + '*)', own.length ? 'Calculados pelo proxy a partir dos bytes que o seu cliente enviou.' : 'Nenhum: a opção fingerprint não está ativa no entrypoint.', own.length ? [t1] : []));

    var t2 = el('table');
    other.forEach(function (k) {
      t2.appendChild(el('tr', null, [el('th', { text: headers[k].name }), el('td', { text: headers[k].value })]));
    });
    root.appendChild(card('Demais cabeçalhos', order.length > 1 ? 'Na ordem em que o navegador os enviou (X-Forwarded-* e similares são do proxy).' : null, [t2]));

    var fs = fullScore(b);
    if (fs) root.appendChild(scoreTable(fs));

    var sections = [
      ['Navegador (navigator)', b.navigator],
      ['Client Hints (userAgentData)', b.userAgentData],
      ['Idioma e fuso (Intl)', b.intl],
      ['Tela', b.screen],
      ['Preferências (media queries)', b.media],
      ['Rede (Network Information API)', b.network],
      ['Navegação (Navigation Timing)', b.navigation],
      ['WebGL', b.webgl],
      ['Recursos', b.features],
      ['Armazenamento', b.storage],
      ['Relógio', b.clock]
    ];
    sections.forEach(function (s) {
      if (s[1]) root.appendChild(card(s[0], null, [kvTable(s[1])]));
    });

    root.appendChild(card('Servidor infopage', null, [kvTable({
      'Nome': data.name,
      'Hostname': data.hostname,
      'IPs': (data.ip || []).join(', ')
    })]));
  }

  // ---------- tabs and actions ----------

  function selectTab(id) {
    ['resumo', 'tecnico'].forEach(function (t) {
      document.getElementById(t).hidden = t !== id;
      document.getElementById('tab-' + t).setAttribute('aria-selected', String(t === id));
    });
    if (location.hash !== '#' + id) history.replaceState(null, '', '#' + id);
  }

  document.getElementById('tab-resumo').addEventListener('click', function () { selectTab('resumo'); });
  document.getElementById('tab-tecnico').addEventListener('click', function () { selectTab('tecnico'); });
  var fromHash = function () { selectTab(location.hash === '#tecnico' ? 'tecnico' : 'resumo'); };
  window.addEventListener('hashchange', fromHash);
  if (location.hash === '#tecnico') fromHash();

  collect().then(function (b) {
    renderResumo(b);
    renderTecnico(b);

    var report = function () {
      return JSON.stringify({ server: data, browser: b, score: fullScore(b) }, null, 2);
    };
    document.getElementById('copy-json').addEventListener('click', function (ev) {
      var btn = ev.currentTarget;
      navigator.clipboard.writeText(report()).then(function () {
        btn.textContent = 'Copiado!';
        setTimeout(function () { btn.textContent = 'Copiar JSON'; }, 1500);
      });
    });
    document.getElementById('download-json').addEventListener('click', function () {
      var a = el('a', { href: URL.createObjectURL(new Blob([report()], { type: 'application/json' })), download: 'infopage-' + (tx('Conn-Id') || Date.now()) + '.json' });
      document.body.appendChild(a);
      a.click();
      a.remove();
    });

    document.getElementById('footer-info').textContent = 'Servido por ' + (data.name || data.hostname || 'infopage') + ' em ' + (data.time || '');
  });
})();
