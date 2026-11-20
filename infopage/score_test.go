package main

import (
	"cmp"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

// The X-Traefix values below come from the goldens of pkg/fingerprint/testdata/golden: the Tls-Hello is what
// Traefix computes from the ClientHello of each capture, the JA4, Akamai and header order are the outputs of the
// FoxIO tool and tshark, the JA4T of the TCP captures. The User-Agent is the one of the measured client, or a
// forged one.
const (
	idTLSKnown  = "TLS-KNOWN"
	idH2Pseudo  = "HTTP-H2-PSEUDO"
	idTCPOS     = "TCP-OS"
	idTLSGrease = "TLS-GREASE"
	alpnBrowser = "h2,http/1.1"

	uaChrome   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36"
	uaHeadless = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/157.0.0.0 Safari/537.36"
	uaFirefox  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:159.0) Gecko/20100101 Firefox/159.0"
	uaIPhone   = "Mozilla/5.0 (iPhone; CPU iPhone OS 26_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.1 Mobile/15E148 Safari/604.1"
	uaBot      = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"

	helloChrome = "legacy_ver=0303,ciphers=g-1301-1302-1303-c02b-c02f-c02c-c030-cca9-cca8-c013-c014-009c-009d-002f-0035," +
		"exts=g-0000-0033-002b-0012-ca34-0023-000b-002d-0005-44cd-fe0d-0010-001b-000d-ff01-0017-000a-g,groups=g-11ec-001d-0017-0018," +
		"key_shares=g-11ec-001d,versions=g-0304-0303,point_formats=0,psk_modes=1,cert_compress=0002,alpn=h2-http/1.1,alps=h2,ech_ext=1"
	helloFirefox = "legacy_ver=0303,ciphers=1301-1303-1302-c02b-c02f-cca9-cca8-c02c-c030-c013-c014-009c-009d-002f-0035," +
		"exts=0000-0017-ff01-000a-000b-0023-0010-0005-0022-0012-0033-002b-000d-002d-001c-001b-fe0d,groups=11ec-001d-0017-0018-0019," +
		"key_shares=11ec-001d-0017,versions=0304-0303,point_formats=0,psk_modes=1,cert_compress=0001-0002-0003,record_limit=16385,alpn=h2-http/1.1,ech_ext=1"
	helloCurl = "legacy_ver=0303,ciphers=1302-1303-1301-c02c-c030-009f-cca9-cca8-ccaa-c02b-c02f-009e-c024-c028-006b-c023-c027-0067-c00a-c014-0039-c009-c013-0033-009d-009c-003d-003c-0035-002f," +
		"exts=ff01-000b-000a-0016-0017-0031-000d-002b-002d-0033-0000-0010,groups=11ec-11eb-11ee-001d-0017-001e-0018-0019-0029-0100-0101," +
		"key_shares=11ec-001d,versions=0304-0303,point_formats=0,psk_modes=1,alpn=h2-http/1.1,ech_ext=0"
	helloGo = "legacy_ver=0303,ciphers=c02b-c02f-c02c-c030-cca9-cca8-c009-c013-c00a-c014-1301-1302-1303," +
		"exts=0000-000b-ff01-0017-0012-0005-000a-000d-0032-0010-002b-0033,groups=11ec-11eb-11ed-001d-0017-0018-0019," +
		"key_shares=11ec-001d,versions=0304-0303,point_formats=0,alpn=h2,ech_ext=0"

	ja4tApple = "65535_2-1-3-1-1-8-4-0-0_1430_6"
	ja4tLinux = "64240_2-4-8-1-3_1460_12"

	orderChrome  = "sec-ch-ua,sec-ch-ua-mobile,sec-ch-ua-platform,upgrade-insecure-requests,user-agent,accept,sec-fetch-site,sec-fetch-mode,sec-fetch-user,sec-fetch-dest,accept-encoding,accept-language,priority"
	orderFirefox = "user-agent,accept,accept-language,accept-encoding,upgrade-insecure-requests,sec-fetch-dest,sec-fetch-mode,sec-fetch-site,priority,te"
)

type client struct {
	ua, ja4, hello, h2, order, ja4t, alpn string
	browserHeaders                        string // "chrome", "firefox", "safari" or ""
	ttlInit                               string
	gapUs, rttUs                          string
	estUs                                 string
}

var (
	chromeH2 = client{
		ua: uaChrome, ja4: "t13d1517h2_8daaf6152771_cb7bf5808d99", hello: helloChrome,
		h2: "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p", order: orderChrome, ja4t: ja4tApple, alpn: alpnBrowser, browserHeaders: "chrome", ttlInit: "64",
	}
	firefoxH2 = client{
		ua: uaFirefox, ja4: "t13d1517h2_8daaf6152771_3cbfd9057e0d", hello: helloFirefox,
		h2: "1:65536;2:0;4:131072;5:16384|12517377|0|m,p,a,s", order: orderFirefox, ja4t: ja4tApple, alpn: alpnBrowser, browserHeaders: engFirefox, ttlInit: "64",
	}
	curlH2 = client{
		ua: "curl/8.23.0", ja4: "t13d3012h2_1d37bd780c83_581742d1f3c4", hello: helloCurl,
		h2: "3:100;4:65536;2:0|1048510465|0|m,s,a,p", order: "user-agent,accept", ja4t: ja4tApple, alpn: alpnBrowser, ttlInit: "64",
	}
	goH2 = client{
		ua: "Go-http-client/2.0", ja4: "t13d1312h2_f57a46bbacb6_f50d94e863eb", hello: helloGo,
		h2: "2:0;4:4194304;5:1048576;6:10485760|1073741824|0|a,m,p,s", order: "accept-encoding,user-agent", ja4t: ja4tApple, alpn: "h2", ttlInit: "64",
	}
)

func newScoreRequest(t *testing.T, c *client, ip string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	h := req.Header
	if c.ua != "" {
		h.Set("User-Agent", c.ua)
	}
	set := func(k, v string) {
		if v != "" {
			h.Set("X-Traefix-"+k, v)
		}
	}
	set("Src-Ip", ip)
	set("Proto", "h2")
	set("Ja4", c.ja4)
	set("Alpn", c.alpn)
	set("Tls", "ver=1.3,cipher=TLS_AES_128_GCM_SHA256,group=X25519MLKEM768,alpn=h2,resumed=0,ech=0")
	set("Tls-Hello", c.hello)
	set("H2", c.h2)
	set("Header-Order", c.order)
	set("Ja4t", c.ja4t)
	if c.ttlInit != "" {
		set("Transport", "proto=tcp,ipv=4,ttl="+c.ttlInit+",ttl_init="+c.ttlInit+",hops=14")
	}
	gap, rtt := c.gapUs, c.rttUs
	if gap == "" {
		gap, rtt = "900", "23000"
	}
	set("Timing", "src=kernel,proto=tcp,tcp_rtt_us="+rtt+",est_to_first_us="+cmp.Or(c.estUs, "300")+",rtt_gap_us="+gap)

	switch c.browserHeaders {
	case "chrome":
		h.Set("Sec-Ch-Ua", `"Chromium";v="156", "Google Chrome";v="156", "Not_A Brand";v="24"`)
		h.Set("Sec-Ch-Ua-Mobile", "?0")
		h.Set("Sec-Ch-Ua-Platform", `"macOS"`)
		fallthrough
	case engFirefox, "safari":
		h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		h.Set("Accept-Encoding", "gzip, deflate, br, zstd")
		h.Set("Accept-Language", "pt-BR,pt;q=0.9")
		h.Set("Sec-Fetch-Site", "none")
		h.Set("Sec-Fetch-Mode", "navigate")
		h.Set("Sec-Fetch-Dest", "document")
	default:
		h.Set("Accept", "*/*")
	}
	return req
}

// chromeSOCKS is Chrome through a SOCKS5 proxy on the host of the Traefix, with the times measured by
// finger-sniffer on finger-01 (task 313): the SYN is the proxy's Linux stack, the TCP round trip is local, the TLS
// round trip and the first byte carry the distance to the client.
var chromeSOCKS = func() client {
	c := chromeH2
	c.ja4t, c.gapUs, c.rttUs, c.estUs = ja4tLinux, "14600", "20", "17500"
	return c
}()

func rule(sc *Score, id string) *scoreResult {
	for i := range sc.Results {
		if sc.Results[i].ID == id {
			return &sc.Results[i]
		}
	}
	return nil
}

func TestScoreClients(t *testing.T) {
	forged := func(c client, ua string) client {
		c.ua = ua
		return c
	}
	chromeBehindCurl := forged(curlH2, uaChrome)
	chromeBehindCurl.browserHeaders = ""

	testCases := []struct {
		desc   string
		c      client
		minR   int
		maxR   int
		role   string
		known  string
		fired  []string
		silent []string
	}{
		{desc: "Chrome", c: chromeH2, maxR: 20, role: roleBrowser, known: knownChrome, silent: []string{idTLSGrease, "TLS-EXT", idH2Pseudo, idTCPOS}},
		{desc: "Firefox", c: firefoxH2, maxR: 20, role: roleBrowser, known: knownFirefox, silent: []string{idTLSGrease, "TLS-EXT", "HTTP-ORDER"}},
		{desc: "curl with Chrome UA", c: chromeBehindCurl, minR: 81, role: roleBrowser, fired: []string{idTLSKnown, "TLS-CIPHERS", idTLSGrease, "HTTP-CH", "HTTP-FETCH", idH2Pseudo}},
		{desc: "Go with Firefox UA", c: forged(goH2, uaFirefox), minR: 81, fired: []string{idTLSKnown, "TLS-CIPHERS", "HTTP-COUNT"}},
		{desc: "Chrome stack with Firefox UA", c: forged(chromeH2, uaFirefox), minR: 51, fired: []string{idTLSKnown, idTLSGrease, "HTTP-CH", idH2Pseudo}},
		{desc: "honest curl", c: curlH2, minR: 81, role: roleAutomation, known: knownCurl},
		{desc: "honest Go", c: goH2, minR: 81, role: roleAutomation, known: knownGo},
		{desc: "headless Chrome", c: forged(chromeH2, uaHeadless), minR: 81, role: roleAutomation},
		{desc: "no User-Agent", c: forged(curlH2, ""), minR: 51, role: roleUnknown},
		{desc: "Chrome UA over a Linux TCP stack", c: func() client { c := chromeH2; c.ja4t = ja4tLinux; return c }(), maxR: 20, fired: []string{idTCPOS}},
		{desc: "Chrome with a slow TLS, ClientHello on time", c: func() client { c := chromeH2; c.gapUs, c.rttUs = "80000", "20000"; return c }(), maxR: 20, fired: []string{idTimeRelay}},
		{desc: "Chrome through a SOCKS proxy", c: chromeSOCKS, minR: 81, role: roleProxy, fired: []string{idTimeRelay}, silent: []string{idTCPOS}},
		{desc: "Chrome with a short gap and a late ClientHello", c: func() client { c := chromeH2; c.gapUs, c.estUs = "6000", "12000"; return c }(), maxR: 20, role: roleBrowser, silent: []string{idTimeRelay}},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			sc := computeScore(t.Context(), newScoreRequest(t, &test.c, "203.0.113.7"), "X-Traefix-", true, nil)
			checkScore(t, sc, test.minR, test.maxR, test.role, test.known)
			checkRules(t, sc, test.fired, test.silent)
		})
	}
}

func TestScoreProxyHasNoRelief(t *testing.T) {
	sc := computeScore(t.Context(), newScoreRequest(t, &chromeSOCKS, "203.0.113.7"), "X-Traefix-", true, nil)
	if r := rule(sc, idTimeRelay); r == nil || r.Points != 60 || r.State != stContradiction {
		t.Fatalf("TIME-RELAY: %+v", r)
	}
	if r := rule(sc, idTCPOS); r == nil || r.Points != 0 {
		t.Errorf("TCP-OS counted with the SYN of the proxy: %+v", r)
	}
	for _, r := range sc.Results {
		if r.Points < 0 {
			t.Errorf("%s relieves a proxied connection: %+d", r.ID, r.Points)
		}
	}
	if sc.Passive.Families[famTime] != 60 {
		t.Errorf("time family = %d", sc.Passive.Families[famTime])
	}
}

func checkScore(t *testing.T, sc *Score, minR, maxR int, role, known string) {
	t.Helper()
	p := sc.Passive
	if p.R < minR || (maxR > 0 && p.R > maxR) {
		t.Errorf("R = %d, want [%d, %d]; results %+v", p.R, minR, maxR, sc.Results)
	}
	if p.Insufficient {
		t.Errorf("coverage %d%%", p.Coverage)
	}
	if role != "" && sc.Role != role {
		t.Errorf("role = %q, want %q", sc.Role, role)
	}
	if known != "" && sc.Client.Known != known {
		t.Errorf("known = %q, want %q", sc.Client.Known, known)
	}
}

func checkRules(t *testing.T, sc *Score, fired, silent []string) {
	t.Helper()
	for _, id := range fired {
		if r := rule(sc, id); r == nil || r.Points <= 0 {
			t.Errorf("%s did not fire: %+v", id, r)
		}
	}
	for _, id := range silent {
		if r := rule(sc, id); r == nil || r.Points > 0 {
			t.Errorf("%s fired or is missing: %+v", id, r)
		}
	}
}

func TestScoreWithoutTraefix(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.Header.Set("User-Agent", uaChrome)
	req.Header.Set("Accept-Language", "pt-BR")

	sc := computeScore(t.Context(), req, "X-Traefix-", true, nil)
	if !sc.Passive.Insufficient || sc.Passive.Band != "Evidência insuficiente" {
		t.Errorf("got %+v", sc.Passive)
	}
}

func TestScoreFamilyCap(t *testing.T) {
	sc := computeScore(t.Context(), newScoreRequest(t, func() *client { c := curlH2; c.ua = uaChrome; return &c }(), "203.0.113.7"), "X-Traefix-", true, nil)
	if got := sc.Passive.Families["tls"]; got != familyCaps["tls"].Max {
		t.Errorf("tls = %d, want the cap %d", got, familyCaps["tls"].Max)
	}
}

func testStore(t *testing.T) *listStore {
	t.Helper()
	s := newListStore()
	s.sets["icloud_relay"] = mustParseList(t, "172.224.226.0/27,BR,BR-SP,São Paulo,\n2a02:26f7::/48,BR,BR-SP,São Paulo,\n", true)
	s.sets["aws"] = mustParseList(t, `{"prefixes":[{"ip_prefix":"3.5.140.0/22"}],"ipv6_prefixes":[{"ipv6_prefix":"2600:1f00::/24"}]}`, false)
	s.sets["googlebot"] = mustParseList(t, `{"prefixes":[{"ipv4Prefix":"66.249.66.0/27"}]}`, false)
	s.sets["gptbot"] = mustParseList(t, `{"prefixes":[{"ipv4Prefix":"20.171.206.0/24"}]}`, false)
	return s
}

func TestScoreNetwork(t *testing.T) {
	store := testStore(t)
	safari := client{
		ua: uaIPhone, ja4: "t13d2014h2_a09f3c656075_14788d8d241b", alpn: alpnBrowser, browserHeaders: "safari",
		hello: "ciphers=g-1301,exts=g-0000-001b-0010-g,groups=g-11ec-001d", order: "user-agent,accept,accept-language,accept-encoding,sec-fetch-site,sec-fetch-mode,sec-fetch-dest,priority,x-a,x-b",
		ja4t: ja4tLinux, ttlInit: "64", gapUs: "60000", rttUs: "10000",
	}

	t.Run("iCloud Private Relay", func(t *testing.T) {
		sc := computeScore(t.Context(), newScoreRequest(t, &safari, "172.224.226.5"), "X-Traefix-", true, store)
		if sc.Network == nil || sc.Network.Relay == "" {
			t.Fatalf("relay not found: %+v", sc.Network)
		}
		for _, id := range []string{idTCPOS, idTimeRelay} {
			if r := rule(sc, id); r == nil || r.Points != 0 {
				t.Errorf("%s: %+v", id, r)
			}
		}
		if sc.Passive.R > 20 {
			t.Errorf("R = %d", sc.Passive.R)
		}
	})

	t.Run("browser from a cloud", func(t *testing.T) {
		sc := computeScore(t.Context(), newScoreRequest(t, &chromeH2, "3.5.141.9"), "X-Traefix-", true, store)
		if r := rule(sc, "NET-CLOUD"); r == nil || r.Points != 20 || r.Value != "Amazon AWS" {
			t.Errorf("NET-CLOUD: %+v", r)
		}
	})

	t.Run("fake Googlebot", func(t *testing.T) {
		c := curlH2
		c.ua = uaBot
		sc := computeScore(t.Context(), newScoreRequest(t, &c, "198.51.100.1"), "X-Traefix-", true, store)
		if sc.Role != roleCrawlerNotVerified || rule(sc, "DECL-UA").Points != 60 {
			t.Errorf("role %q, %+v", sc.Role, rule(sc, "DECL-UA"))
		}
	})

	t.Run("GPTBot from its list", func(t *testing.T) {
		c := curlH2
		c.ua = "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; GPTBot/1.2; +https://openai.com/gptbot)"
		sc := computeScore(t.Context(), newScoreRequest(t, &c, "20.171.206.10"), "X-Traefix-", true, store)
		if sc.Role != roleCrawlerVerified || sc.Passive.R > 20 {
			t.Errorf("role %q, R %d", sc.Role, sc.Passive.R)
		}
	})
}

func TestParseUA(t *testing.T) {
	testCases := []struct {
		ua      string
		engine  string
		os      string
		tool    string
		crawler string
	}{
		{ua: uaChrome, engine: engChromium, os: osMacOS},
		{ua: uaFirefox, engine: engFirefox, os: osMacOS},
		{ua: uaIPhone, engine: engWebkit, os: "ios"},
		{ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 26_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/156.0 Mobile/15E148 Safari/604.1", engine: engWebkit, os: "ios"},
		{ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36 Edg/156.0.0.0", engine: engChromium, os: "windows"},
		{ua: "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Mobile Safari/537.36", engine: engChromium, os: "android"},
		{ua: uaHeadless, engine: engChromium, os: osMacOS, tool: "HeadlessChrome"},
		{ua: "python-requests/2.32.3", tool: "python-requests"},
		{ua: "curl/8.7.1", tool: "curl"},
		{ua: uaBot, crawler: "googlebot"},
		{ua: "Mozilla/5.0 (compatible; SomethingBot/1.0)", crawler: "other"},
	}
	for _, test := range testCases {
		c := parseUA(test.ua)
		if c.Engine != test.engine || c.OS != test.os || c.Tool != test.tool || c.Crawler != test.crawler {
			t.Errorf("%s: got %+v", test.ua, c)
		}
	}
}

func mustParseList(t *testing.T, body string, csv bool) *rangeSet {
	t.Helper()
	set, err := parseList(strings.NewReader(body), csv)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestRangeSet(t *testing.T) {
	set := mustParseList(t, `{"a":["10.0.0.0/8","10.1.0.0/16"],"b":{"c":"2001:db8::/32","d":"192.0.2.1"},"n":3}`, false)
	if set.count != 3 {
		t.Fatalf("count = %d", set.count)
	}
	for ip, want := range map[string]bool{"10.255.255.255": true, "11.0.0.0": false, "2001:db8:ffff::1": true, "2001:db9::": false, "192.0.2.1": true, "192.0.2.2": false} {
		if _, got := set.lookup(netip.MustParseAddr(ip)); got != want {
			t.Errorf("%s: got %v", ip, got)
		}
	}
	if !hasDomainSuffix("crawl-66-249-66-1.googlebot.com", []string{"googlebot.com"}) || hasDomainSuffix("evilgooglebot.com", []string{"googlebot.com"}) {
		t.Error("suffix match")
	}
}

func TestScoreText(t *testing.T) {
	req := newScoreRequest(t, &curlH2, "203.0.113.7")
	rw := httptest.NewRecorder()
	infopageHandler(rw, req)
	if !strings.Contains(rw.Body.String(), "Automation-Score: ") || !strings.Contains(rw.Body.String(), roleAutomation) {
		t.Errorf("got %q", rw.Body.String())
	}
}
