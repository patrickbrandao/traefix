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
        viewport: window.innerWidth + 'x' + window.innerHeight
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

    var mss = num(transport.syn_mss);
    if (mss !== undefined) {
      var full = v6 ? 1440 : 1460;
      if (mss === full) add('ok', 'Tamanho do pacote (MSS)', mss + ' bytes', 'O máximo de uma rede Ethernet/Wi-Fi comum (MTU 1500).');
      else if (mss === full - 8) add('info', 'Tamanho do pacote (MSS)', mss + ' bytes', 'Típico de PPPoE (fibra e DSL). Diferença de velocidade desprezível.');
      else if (mss > full) add('info', 'Tamanho do pacote (MSS)', mss + ' bytes', 'Acima do padrão: rede com jumbo frames ou configuração manual.');
      else if (mss >= (v6 ? 1220 : 1200)) add('info', 'Tamanho do pacote (MSS)', mss + ' bytes', 'Menor que o padrão: túnel, VPN ou rede móvel no caminho. A perda de velocidade é de 1% a 3%.');
      else add('warn', 'Tamanho do pacote (MSS)', mss + ' bytes', 'Muito pequeno: pilha de rede limitada ou ajuste de MSS incorreto.');
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

    // One connection is noisy (slow certificate check, SYN proxy before Traefix): only a large gap counts,
    // and it stays a hint, never an alert.
    var gap = Number(timing.rtt_gap_us), rtt = Number(timing.tcp_rtt_us);
    if (isFinite(gap) && isFinite(rtt) && timing.rtt_gap_us !== undefined && timing.tcp_rtt_us !== undefined) {
      if (gap > 30000 && gap > rtt * 0.5) add('warn', 'O TLS levou ' + ms(gap) + ' a mais que o TCP: possível proxy ou relay no caminho (SOCKS, CONNECT). Uma única conexão pode enganar.');
      else add('ok', 'Sem sinal de proxy TCP no caminho (tempos de TCP e TLS coerentes).');
    }

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
      return JSON.stringify({ server: data, browser: b }, null, 2);
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
