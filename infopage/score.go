package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// The automation score of infopage/SCORE.md: each rule compares what the client declares (User-Agent, Client
// Hints) with what the lower layers show (TLS, HTTP/2, header order, TCP, timing, address lists), and gives
// points per causal family, capped, so that ten hints from the same ClientHello count as one. The index is a
// heuristic, not a calibrated probability.

const scoreVersion = "score/v1"

// idTimeRelay is the rule of a proxy or SOCKS seen by the timing of the connection.
const idTimeRelay = "TIME-RELAY"

// Rule states.
const (
	stIdeal         = "ideal"
	stNormal        = "normal"
	stAtypical      = "atipico"
	stContradiction = "contradicao"
	stDeclared      = "declarado"
	stUnknown       = "desconhecido"
)

// Families.
const (
	famDecl = "decl"
	famTLS  = "tls"
	famHTTP = "http"
	famTCP  = "tcp"
	famTime = "time"
	famNet  = "net"
	famJS   = "js"
)

// Engines, systems and stacks.
const (
	engChromium = "chromium"
	engFirefox  = "firefox"
	engWebkit   = "webkit"

	osWindows  = "windows"
	osMacOS    = "macos"
	osIOS      = "ios"
	osAndroid  = "android"
	osLinux    = "linux"
	osChromeOS = "chromeos"
	stackApple = "apple"
)

// Roles.
const (
	roleBrowser    = "navegador"
	roleAutomation = "automacao_declarada"
	// roleProxy is a browser whose connection crossed a proxy or SOCKS (TIME-RELAY confirmed).
	roleProxy              = "via_proxy"
	roleCrawlerVerified    = "crawler_verificado"
	roleCrawlerNotVerified = "crawler_nao_verificado"
	roleUnknown            = "desconhecido"
)

// Robots with a published address list.
const (
	robotGoogle        = "googlebot"
	robotBing          = "bingbot"
	robotApple         = "applebot"
	robotGPT           = "gptbot"
	robotOAISearch     = "oai-searchbot"
	robotChatGPT       = "chatgpt-user"
	robotDuckDuckGo    = "duckduckbot"
	robotPerplexity    = "perplexitybot"
	robotPerplexityUsr = "perplexity-user"
	robotCommonCrawl   = "ccbot"
)

const (
	tokChrome  = "Chrome/"
	nameChrome = "Chrome"
	txtYes     = "sim"
	txtNo      = "não"
	txtMissing = "ausente"
	protoH2    = "h2"
)

// Score calibration, see SCORE.md: S = 0 gives 10.
const (
	scoreA = 0.07
	scoreB = -2.2
)

type familyCap struct {
	Max int `json:"max"`
	Min int `json:"min"`
}

var familyCaps = map[string]familyCap{
	famDecl: {Max: 60},
	famTLS:  {Max: 45, Min: -15},
	famHTTP: {Max: 40, Min: -15},
	famTCP:  {Max: 30, Min: -10},
	famTime: {Max: 60},
	famNet:  {Max: 45},
	famJS:   {Max: 40, Min: -10},
}

type scoreResult struct {
	ID       string `json:"id"`
	Family   string `json:"family"`
	Name     string `json:"name"`
	Value    string `json:"value,omitempty"`
	Expected string `json:"expected,omitempty"`
	State    string `json:"state"`
	Points   int    `json:"points"`
	// Weight is what the rule can weigh: the coverage is the weight of the rules evaluated over the weight of
	// the rules that apply.
	Weight int    `json:"weight"`
	Reason string `json:"reason,omitempty"`
	Benign string `json:"benign,omitempty"`
}

type clientClaim struct {
	Engine  string `json:"engine,omitempty"`
	Browser string `json:"browser,omitempty"`
	Version int    `json:"version,omitempty"`
	OS      string `json:"os,omitempty"`
	Mobile  bool   `json:"mobile,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Crawler string `json:"crawler,omitempty"`
	// Known is the measured client whose TLS fingerprint equals the one of the connection.
	Known string `json:"known,omitempty"`
}

type scoreSummary struct {
	S            int            `json:"s"`
	R            int            `json:"r"`
	Band         string         `json:"band"`
	Coverage     int            `json:"coverage"`
	Insufficient bool           `json:"insufficient,omitempty"`
	Families     map[string]int `json:"families"`
}

// Score is the passive part of the score, computed from the request; the page adds the rules of the browser
// APIs (family js) with the same caps and calibration.
type Score struct {
	Version  string               `json:"version"`
	Client   clientClaim          `json:"client"`
	Role     string               `json:"role"`
	IP       string               `json:"ip,omitempty"`
	Network  *netMatch            `json:"network,omitempty"`
	Results  []scoreResult        `json:"results"`
	Families map[string]familyCap `json:"families"`
	A        float64              `json:"a"`
	B        float64              `json:"b"`
	Passive  scoreSummary         `json:"passive"`
}

// knownClient is a client measured in the lab: the base of finger-sniffer (internal/bot/lists/known_clients.json,
// macOS, 2026-11-16) and the goldens of pkg/fingerprint/testdata/golden.
type knownClient struct {
	Name    string
	Engine  string
	JA4     string
	Library bool
}

const (
	knownChrome     = "Chrome 157"
	knownFirefox    = "Firefox 159"
	knownCurl       = "curl 8.23 (OpenSSL)"
	knownCurlLibre  = "curl 8.7 (LibreSSL)"
	knownGo         = "Go 1.27"
	knownPythonReqs = "python-requests 2.32"
)

var knownClients = []knownClient{
	{Name: knownChrome, Engine: engChromium, JA4: "t13d1516h1_8daaf6152771_5b37b36a67c0"},
	{Name: knownChrome, Engine: engChromium, JA4: "t13d1517h2_8daaf6152771_cb7bf5808d99"},
	{Name: knownChrome, Engine: engChromium, JA4: "q13d0312h3_55b375c5d22e_178839b6cec1"},
	{Name: knownFirefox, Engine: engFirefox, JA4: "t13d1517h1_8daaf6152771_3cbfd9057e0d"},
	{Name: knownFirefox, Engine: engFirefox, JA4: "t13d1517h2_8daaf6152771_3cbfd9057e0d"},
	{Name: knownFirefox, Engine: engFirefox, JA4: "q13d0315h3_55b375c5d22e_dc5437974b47"},
	{Name: knownCurl, JA4: "t13d3012h1_1d37bd780c83_581742d1f3c4", Library: true},
	{Name: knownCurl, JA4: "t13d3012h2_1d37bd780c83_581742d1f3c4", Library: true},
	{Name: knownCurl, JA4: "q13d0311h3_55b375c5d22e_df5ee7f3ca36", Library: true},
	{Name: knownCurlLibre, JA4: "t13d4907h1_0d8feac7bc37_7395dae3b2f3", Library: true},
	{Name: knownCurlLibre, JA4: "t13d4907h2_0d8feac7bc37_7395dae3b2f3", Library: true},
	{Name: knownGo, JA4: "t13d131100_f57a46bbacb6_a089bac06eae", Library: true},
	{Name: knownGo, JA4: "t13d131100_f57a46bbacb6_f50d94e863eb", Library: true},
	{Name: knownGo, JA4: "t13d1312h2_f57a46bbacb6_a089bac06eae", Library: true},
	{Name: knownGo, JA4: "t13d1312h2_f57a46bbacb6_f50d94e863eb", Library: true},
	{Name: knownGo, JA4: "q13d0313h3_55b375c5d22e_4156cdf64688", Library: true},
	{Name: knownPythonReqs, JA4: "t12d3805h1_10ed599f3404_aaf95bb78ec9", Library: true},
}

// libraryCipherHashes are the JA4 cipher hashes (part b) of the default lists of the TLS libraries: a browser
// never offers them. Chromium and Firefox share 8daaf6152771, so part b alone doesn't tell them apart.
var libraryCipherHashes = map[string]string{
	"1d37bd780c83": "OpenSSL (curl 8.23)",
	"0d8feac7bc37": "LibreSSL (curl 8.7)",
	"f57a46bbacb6": "Go crypto/tls",
	"10ed599f3404": "Python ssl (requests)",
}

type engineRef struct {
	Pseudo   string
	Settings string
	Window   string
}

// h2Refs are the Akamai HTTP/2 values of the goldens. WebKit has no reference yet.
var h2Refs = map[string]engineRef{
	engChromium: {Pseudo: "m,a,s,p", Settings: "1:65536;2:0;4:6291456;6:262144", Window: "15663105"},
	engFirefox:  {Pseudo: "m,p,a,s", Settings: "1:65536;2:0;4:131072;5:16384", Window: "12517377"},
}

var h2Libraries = map[string]string{"m,s,a,p": "curl (nghttp2)", "a,m,p,s": "Go net/http"}

var (
	automationRe = regexp.MustCompile(`(?i)HeadlessChrome|PhantomJS|SlimerJS|Selenium|WebDriver|Puppeteer|Playwright|Cypress|HtmlUnit`)
	crawlerRe    = regexp.MustCompile(`(?i)crawler|spider|scraper|bot[/;)]`)
	versionRe    = regexp.MustCompile(`^(\d+)`)
)

var libraryPrefixes = []struct{ prefix, name string }{
	{"curl/", "curl"},
	{"wget/", "Wget"},
	{"python-requests/", "python-requests"},
	{"python-urllib/", "Python urllib"},
	{"python-httpx/", "httpx"},
	{"aiohttp/", "aiohttp"},
	{"python/", "Python"},
	{"go-http-client/", "Go net/http"},
	{"okhttp/", "OkHttp"},
	{"java/", "Java"},
	{"java-http-client/", "Java HttpClient"},
	{"apache-httpclient/", "Apache HttpClient"},
	{"libwww-perl/", "libwww-perl"},
	{"node-fetch", "node-fetch"},
	{"axios/", "axios"},
	{"undici", "undici"},
	{"node", "Node.js"},
	{"scrapy/", "Scrapy"},
	{"httpie/", "HTTPie"},
	{"postmanruntime/", "Postman"},
	{"insomnia/", "Insomnia"},
	{"guzzlehttp/", "Guzzle"},
	{"ruby", "Ruby"},
	{"dart:io", "Dart"},
	{"reqwest", "reqwest"},
	{"powershell", "PowerShell"},
	{"winhttp", "WinHTTP"},
	{"libcurl", "libcurl"},
	{"faraday", "Faraday"},
}

// crawlers maps a User-Agent token to the robot; the robots with a published list are verified by it.
var crawlers = []struct{ token, robot string }{
	{"googlebot", robotGoogle},
	{"adsbot-google", robotGoogle},
	{"mediapartners-google", robotGoogle},
	{"storebot-google", robotGoogle},
	{"google-inspectiontool", robotGoogle},
	{"googleother", robotGoogle},
	{"google-extended", robotGoogle},
	{"apis-google", robotGoogle},
	{"feedfetcher-google", robotGoogle},
	{"bingbot", robotBing},
	{"bingpreview", robotBing},
	{"adidxbot", robotBing},
	{"applebot", robotApple},
	{"gptbot", robotGPT},
	{"oai-searchbot", robotOAISearch},
	{"chatgpt-user", robotChatGPT},
	{"duckduckbot", robotDuckDuckGo},
	{"duckassistbot", robotDuckDuckGo},
	{"perplexitybot", robotPerplexity},
	{"perplexity-user", robotPerplexityUsr},
	{"ccbot", robotCommonCrawl},
	{"yandex", "yandex"},
	{"baiduspider", "baidu"},
	{"ahrefsbot", "ahrefs"},
	{"semrushbot", "semrush"},
	{"mj12bot", "majestic"},
	{"dotbot", "moz"},
	{"petalbot", "petal"},
	{"bytespider", "bytespider"},
	{"claudebot", "claudebot"},
	{"claude-user", "claude-user"},
	{"claude-searchbot", "claude-searchbot"},
	{"facebookexternalhit", "facebook"},
	{"meta-externalagent", "facebook"},
	{"twitterbot", "twitter"},
	{"linkedinbot", "linkedin"},
	{"slackbot", "slack"},
	{"discordbot", "discord"},
	{"telegrambot", "telegram"},
	{"whatsapp", "whatsapp"},
	{"amazonbot", "amazon"},
	{"seznambot", "seznam"},
}

// browserTokens are tried in order: Edge, Opera and Samsung also carry "Chrome/".
var browserTokens = []struct{ token, engine, browser, version string }{
	{"Firefox/", engFirefox, "Firefox", "Firefox/"},
	{"Edg/", engChromium, "Edge", "Edg/"},
	{"OPR/", engChromium, "Opera", tokChrome},
	{"SamsungBrowser/", engChromium, "Samsung Internet", tokChrome},
	{"HeadlessChrome/", engChromium, nameChrome, "HeadlessChrome/"},
	{tokChrome, engChromium, nameChrome, tokChrome},
}

var iosBrowsers = []struct{ token, browser string }{
	{"CriOS/", nameChrome}, {"FxiOS/", "Firefox"}, {"EdgiOS/", "Edge"},
}

func majorVersion(s string) int {
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func after(ua, token string) string {
	_, rest, _ := strings.Cut(ua, token)
	return rest
}

func uaOS(ua string) string {
	switch {
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"), strings.Contains(ua, "iPod"):
		return osIOS
	case strings.Contains(ua, "Windows"):
		return osWindows
	case strings.Contains(ua, "Android"):
		return osAndroid
	case strings.Contains(ua, "CrOS"):
		return osChromeOS
	case strings.Contains(ua, "Mac OS X"), strings.Contains(ua, "Macintosh"):
		return osMacOS
	case strings.Contains(ua, "Linux"):
		return osLinux
	}
	return ""
}

// uaProgram recognizes the HTTP libraries and the robots, which declare themselves.
func uaProgram(c *clientClaim, ua string) {
	lower := strings.ToLower(ua)
	for _, l := range libraryPrefixes {
		if strings.HasPrefix(lower, l.prefix) {
			c.Tool = l.name
			return
		}
	}
	for _, cr := range crawlers {
		if strings.Contains(lower, cr.token) {
			c.Crawler = cr.robot
			return
		}
	}
	if crawlerRe.MatchString(ua) {
		c.Crawler = "other"
	}
}

func uaBrowser(c *clientClaim, ua string) {
	if c.OS == osIOS {
		// Every browser on iOS uses WebKit.
		c.Engine, c.Browser, c.Version = engWebkit, "Safari", majorVersion(after(ua, "Version/"))
		for _, b := range iosBrowsers {
			if strings.Contains(ua, b.token) {
				c.Browser, c.Version = b.browser, majorVersion(after(ua, b.token))
			}
		}
		return
	}
	for _, b := range browserTokens {
		if strings.Contains(ua, b.token) {
			c.Engine, c.Browser, c.Version = b.engine, b.browser, majorVersion(after(ua, b.version))
			return
		}
	}
	if strings.Contains(ua, "Safari/") && strings.Contains(ua, "Version/") {
		c.Engine, c.Browser, c.Version = engWebkit, "Safari", majorVersion(after(ua, "Version/"))
	}
}

func parseUA(ua string) clientClaim {
	c := clientClaim{OS: uaOS(ua), Mobile: strings.Contains(ua, "Mobile") || strings.Contains(ua, "iPhone")}

	uaProgram(&c, ua)
	if c.Tool != "" || c.Crawler != "" {
		return c
	}

	c.Tool = automationRe.FindString(ua)
	if strings.HasPrefix(ua, "Mozilla/5.0") {
		uaBrowser(&c, ua)
	}
	return c
}

// osStack groups the systems by their TCP stack.
func osStack(os string) string {
	switch os {
	case osIOS, osMacOS:
		return stackApple
	case osAndroid, osLinux, osChromeOS:
		return osLinux
	case osWindows:
		return osWindows
	}
	return ""
}

var stackNames = map[string]string{
	stackApple: "Apple (macOS/iOS)", osLinux: "Linux/Android", osWindows: "Windows",
}

// synStack recognizes the SYN of the three stacks the goldens hold (tcp/*.ja4.jsonl and the FoxIO Windows
// sample): the order of the options, and the unscaled window for Apple.
func synStack(win, opts string) string {
	switch opts {
	case "2-1-3-1-1-8-4-0-0":
		if win == "65535" {
			return stackApple
		}
	case "2-4-8-1-3":
		return osLinux
	case "2-1-3-1-1-4":
		return osWindows
	}
	return ""
}

func kv(s string) map[string]string {
	out := make(map[string]string)
	if s == "" {
		return out
	}
	for p := range strings.SplitSeq(s, ",") {
		k, v, ok := strings.Cut(p, "=")
		if ok {
			out[k] = v
		}
	}
	return out
}

func atoi(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func yesNo(b bool) string {
	if b {
		return txtYes
	}
	return txtNo
}

// clientIP is the client address Traefix saw, before the address of the socket, which is the proxy.
func clientIP(r *http.Request, pfx string) netip.Addr {
	forwarded, _, _ := strings.Cut(r.Header.Get("X-Forwarded-For"), ",")
	for _, s := range []string{r.Header.Get(pfx + "Src-Ip"), r.Header.Get("X-Real-IP"), strings.TrimSpace(forwarded)} {
		if a, err := netip.ParseAddr(s); err == nil {
			return a.Unmap()
		}
	}
	if ap, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		return ap.Addr().Unmap()
	}
	return netip.Addr{}
}

type scorer struct {
	r       *http.Request
	pfx     string
	nav     bool
	c       clientClaim
	browser bool
	results []scoreResult
}

func (s *scorer) tx(name string) string { return s.r.Header.Get(s.pfx + name) }

func (s *scorer) add(r *scoreResult) { s.results = append(s.results, *r) }

func (s *scorer) unknown(id, family, name string, weight int, reason string) {
	s.add(&scoreResult{ID: id, Family: family, Name: name, State: stUnknown, Weight: weight, Reason: reason})
}

// computeScore runs the passive rules. nav tells that the request is the navigation of the page, so that its
// Sec-Fetch values are known in advance.
func computeScore(ctx context.Context, r *http.Request, pfx string, nav bool, store *listStore) *Score {
	ua := r.Header.Get("User-Agent")
	s := &scorer{r: r, pfx: pfx, nav: nav, c: parseUA(ua)}
	s.browser = s.c.Engine != "" && s.c.Crawler == ""

	sc := &Score{Version: scoreVersion, Families: familyCaps, A: scoreA, B: scoreB}

	ip := clientIP(r, pfx)
	if ip.IsValid() {
		sc.IP = ip.String()
	}
	var m netMatch
	if store != nil {
		m = store.match(ip)
		sc.Network = &m
	}

	sc.Role = s.declaration(ctx, ua, ip, store, &m)
	s.known()
	if s.browser {
		s.tlsRules()
		s.httpRules()
	}
	s.tcpRules(&m)
	relay := s.timeRules(&m)
	s.netRules(store, &m, sc.Role)

	// A browser that declares its automation is coherent by construction: the reliefs only help who claims to
	// be an ordinary browser.
	if sc.Role == roleAutomation {
		s.noRelief(" Sem alívio: automação declarada.")
	}
	// Through a proxy, the SYN is the proxy's: the TCP rules describe it, not the client, and would count the same
	// cause twice. A coherent browser behind a proxy is exactly the case of a client spreading requests over
	// addresses, so the coherence does not relieve it either.
	if relay {
		for i := range s.results {
			if s.results[i].Family == famTCP && s.results[i].Points > 0 {
				s.results[i].Points = 0
				s.results[i].State = stNormal
				s.results[i].Reason += " Não conta: o SYN é do proxy (TIME-RELAY)."
			}
		}
		s.noRelief(" Sem alívio: proxy ou SOCKS no caminho.")
		if sc.Role == roleBrowser {
			sc.Role = roleProxy
		}
	}

	sc.Client = s.c
	sc.Results = s.results
	sc.Passive = aggregate(s.results)
	return sc
}

func (s *scorer) declaration(ctx context.Context, ua string, ip netip.Addr, store *listStore, m *netMatch) string {
	res := scoreResult{ID: "DECL-UA", Family: famDecl, Name: "Declaração (User-Agent)", Value: ua}
	role := roleBrowser

	switch {
	case ua == "":
		res.Value, res.State, res.Points, res.Weight = txtMissing, stDeclared, 40, 40
		res.Reason = "Sem User-Agent: todo navegador envia um."
		role = roleUnknown
	case s.c.Crawler != "":
		role = s.crawlerDecl(ctx, &res, ip, store, m)
	case s.c.Tool != "" && s.c.Engine == "":
		role = roleAutomation
		res.State, res.Points, res.Weight = stDeclared, 60, 60
		res.Reason = "Cliente HTTP de programa (" + s.c.Tool + "): automação declarada, o que é honesto, não fraude."
	case s.c.Tool != "":
		role = roleAutomation
		res.State, res.Points, res.Weight = stDeclared, 60, 60
		res.Reason = "Navegador sob automação declarada no UA (" + s.c.Tool + ")."
		res.Benign = "Testes automatizados do próprio dono do site."
	case s.c.Engine != "":
		res.State = stIdeal
		res.Expected = "UA de navegador"
		res.Reason = s.c.Browser + " " + strconv.Itoa(s.c.Version) + ": as camadas de baixo são comparadas com essa declaração."
	default:
		role = roleUnknown
		res.State, res.Points, res.Weight = stAtypical, 10, 10
		res.Reason = "UA não reconhecido como navegador nem como programa conhecido."
	}
	s.add(&res)
	return role
}

// crawlerDecl verifies a robot by the list its operator publishes, and by its reverse DNS when the operator
// publishes the suffixes.
func (s *scorer) crawlerDecl(ctx context.Context, res *scoreResult, ip netip.Addr, store *listStore, m *netMatch) string {
	res.State, res.Points, res.Weight = stDeclared, 40, 40
	res.Reason = "Robô declarado (" + s.c.Crawler + "), sem lista oficial para verificar o endereço."
	res.Benign = "Robôs de busca legítimos se declaram; a verificação separa os verdadeiros."
	if !store.listLoaded(s.c.Crawler) {
		return roleCrawlerNotVerified
	}

	inList := slices.Contains(m.Crawlers, s.c.Crawler)
	_, needsDNS := robotSuffixes[s.c.Crawler]
	dnsOK, host := true, ""
	if inList && needsDNS {
		host, dnsOK = store.verifyDNS(ctx, s.c.Crawler, ip)
	}

	res.Points, res.Weight = 60, 60
	switch {
	case inList && dnsOK:
		res.State, res.Points = stIdeal, 0
		res.Reason = "Robô verificado: o endereço está na lista oficial de " + m.CrawlerLabel
		if host != "" {
			res.Reason += " e o DNS reverso (" + host + ") confere"
		}
		res.Reason += ". Robô declarado e verificado, não é fraude."
		return roleCrawlerVerified
	case inList:
		res.Reason = "UA de robô (" + s.c.Crawler + ") com endereço da lista, mas o DNS reverso não confere."
	default:
		res.Reason = "UA de robô (" + s.c.Crawler + ") fora da lista oficial de endereços: robô falso."
	}
	return roleCrawlerNotVerified
}

// ja4Parts splits a JA4 in its a, b and c parts.
func ja4Parts(ja4 string) []string {
	p := strings.Split(ja4, "_")
	if len(p) != 3 || len(p[0]) != 10 {
		return nil
	}
	return p
}

func findKnown(ja4 string) *knownClient {
	for i := range knownClients {
		if knownClients[i].JA4 == ja4 {
			return &knownClients[i]
		}
	}
	return nil
}

// known compares the JA4 with the measured clients.
func (s *scorer) known() {
	ja4 := s.tx("Ja4")
	if ja4 == "" {
		if s.browser {
			s.unknown("TLS-KNOWN", famTLS, "TLS conhecido", 45, "Sem JA4: conexão sem TLS ou fingerprint desligado.")
		}
		return
	}
	hit := findKnown(ja4)
	if hit != nil {
		s.c.Known = hit.Name
	}
	if !s.browser {
		return
	}

	res := scoreResult{
		ID: "TLS-KNOWN", Family: famTLS, Name: "TLS conhecido", Value: ja4, Weight: 45,
		Expected: "JA4 de um " + s.c.Browser + " medido, ou não catalogado",
	}
	lib := ""
	if p := ja4Parts(ja4); p != nil {
		lib = libraryCipherHashes[p[1]]
	}
	switch {
	case hit != nil && hit.Library:
		res.State, res.Points = stContradiction, 45
		res.Reason = "TLS idêntico ao de " + hit.Name + " com UA de " + s.c.Browser + "."
		res.Benign = "Proxy que termina o TLS e reescreve o UA."
	case hit != nil && hit.Engine != s.c.Engine:
		res.State, res.Points = stContradiction, 40
		res.Reason = "TLS idêntico ao de " + hit.Name + ", outro navegador que o declarado (" + s.c.Browser + ")."
		res.Benign = "Extensão que troca o UA."
	case hit != nil:
		res.State, res.Points = stIdeal, -15
		res.Reason = "TLS idêntico ao de " + hit.Name + ", medido em laboratório. Navegador automatizado tem o mesmo."
	case lib != "":
		res.State, res.Points = stContradiction, 30
		res.Reason = "Cifras na ordem padrão de " + lib + ", não de navegador."
		res.Benign = "Proxy de inspeção TLS."
	default:
		res.State = stNormal
		res.Reason = "JA4 fora da base medida (só macOS, Chrome 157 e Firefox 159): versão ou sistema diferente, não é suspeito."
	}
	s.add(&res)
}

func (s *scorer) tlsRules() {
	tls := kv(s.tx("Tls"))
	ja4 := s.tx("Ja4")
	parts := ja4Parts(ja4)
	quic := strings.HasPrefix(ja4, "q") || tls["transport"] == "quic"

	s.tlsVersion(tls["ver"])
	if parts != nil && !quic {
		s.tlsCiphers(parts[0])
	}
	alpn := s.tlsALPN(parts != nil, quic)
	if len(parts) == 3 && parts[0][3] == 'i' {
		s.add(&scoreResult{
			ID: "TLS-SNI", Family: famTLS, Name: "Nome do site (SNI)", Value: txtMissing, Expected: "presente", Weight: 5,
			State: stAtypical, Points: 5, Reason: "Sem SNI: acesso pelo endereço IP.", Benign: "Acesso direto por IP.",
		})
	}
	if quic {
		return
	}

	hello := kv(s.tx("Tls-Hello"))
	if len(hello) == 0 {
		s.unknown("TLS-HELLO", famTLS, "Estrutura do ClientHello", 20, "Sem X-Traefix-Tls-Hello (headerDetail desligado).")
		return
	}
	exts := strings.Split(hello["exts"], "-")
	s.tlsGrease(hello, exts)
	s.tlsExtensions(hello["exts"], exts, alpn)
	s.tlsPQ(hello["groups"])
	s.tlsECH(hello["ech_ext"] == "1")
}

func (s *scorer) tlsVersion(v string) {
	if v == "" {
		s.unknown("TLS-VER", famTLS, "Versão do TLS", 20, "Sem X-Traefix-Tls.")
		return
	}
	r := scoreResult{ID: "TLS-VER", Family: famTLS, Name: "Versão do TLS", Value: v, Expected: "1.3", Weight: 20, State: stIdeal}
	if v != "1.3" {
		r.State, r.Points = stContradiction, 20
		r.Reason = "Navegadores atuais negociam TLS 1.3 com este servidor."
		r.Benign = "Proxy TLS antigo."
	}
	s.add(&r)
}

func (s *scorer) tlsCiphers(a string) {
	n, _ := atoi(a[4:6])
	r := scoreResult{ID: "TLS-CIPHERS", Family: famTLS, Name: "Quantidade de cifras", Value: strconv.Itoa(n), Expected: "15 a 20", Weight: 20, State: stIdeal}
	switch {
	case n > 25 || n < 15:
		r.State, r.Points = stContradiction, 20
		r.Reason = "Navegadores oferecem de 15 a 20 cifras; curl/OpenSSL oferece 30, LibreSSL 49, Go 13."
		r.Benign = "Proxy de inspeção TLS."
	case n > 20:
		r.State, r.Points = stAtypical, 5
	}
	s.add(&r)
}

func (s *scorer) tlsALPN(haveJA4, quic bool) string {
	alpn := s.tx("Alpn")
	if alpn == "" {
		alpn = strings.ReplaceAll(kv(s.tx("Tls-Hello"))["alpn"], "-", ",")
	}
	if alpn == "" && !haveJA4 {
		return ""
	}
	want := protoH2
	if quic {
		want = "h3"
	}
	r := scoreResult{ID: "TLS-ALPN", Family: famTLS, Name: "ALPN oferecido", Value: alpn, Expected: "com " + want, Weight: 30, State: stIdeal}
	if alpn == "" {
		r.Value = "nenhum"
	}
	if !slices.Contains(strings.Split(alpn, ","), want) {
		r.State, r.Points = stContradiction, 30
		r.Reason = "Navegadores atuais sempre oferecem " + want + "."
		r.Benign = "Navegador com HTTP/2 desligado, proxy que quebra o ALPN."
	}
	s.add(&r)
	return alpn
}

func (s *scorer) tlsGrease(hello map[string]string, exts []string) {
	grease := strings.Contains("-"+hello["ciphers"]+"-", "-g-") || slices.Contains(exts, "g")
	want := s.c.Engine != engFirefox
	r := scoreResult{ID: "TLS-GREASE", Family: famTLS, Name: "GREASE", Value: yesNo(grease), Expected: yesNo(want), Weight: 20, State: stIdeal}
	if grease != want {
		r.State, r.Points = stContradiction, 20
		if want {
			r.Reason = s.c.Browser + " envia valores GREASE no ClientHello; bibliotecas não."
		} else {
			r.Reason = "Firefox não envia GREASE; Chromium e bibliotecas que o imitam enviam."
		}
		r.Benign = "Proxy TLS que reescreve o ClientHello."
	}
	s.add(&r)
}

func (s *scorer) tlsExtensions(raw string, exts []string, alpn string) {
	has := func(e string) bool { return slices.Contains(exts, e) }
	alps := has("44cd") || has("4469")
	chromium, firefox := s.c.Engine == engChromium, s.c.Engine == engFirefox

	var bad []string
	if !has("001b") {
		bad = append(bad, "sem compress_certificate (todo navegador envia)")
	}
	if !chromium && alps {
		bad = append(bad, "ALPS, extensão só do Chromium")
	}
	if !firefox && has("001c") {
		bad = append(bad, "record_size_limit, extensão só do Firefox")
	}
	if firefox && !has("001c") {
		bad = append(bad, "sem record_size_limit do Firefox")
	}
	if chromium && strings.Contains(alpn, protoH2) && !alps {
		bad = append(bad, "sem ALPS do Chromium")
	}

	r := scoreResult{
		ID: "TLS-EXT", Family: famTLS, Name: "Extensões do motor", Value: raw, Weight: 20, State: stIdeal,
		Expected: map[string]string{engChromium: "compress_certificate e ALPS", engFirefox: "compress_certificate e record_size_limit", engWebkit: "compress_certificate"}[s.c.Engine],
	}
	if len(bad) > 0 {
		r.State, r.Points = stContradiction, 20
		r.Reason = strings.Join(bad, "; ") + "."
		r.Benign = "Proxy TLS."
	}
	s.add(&r)
}

func (s *scorer) tlsPQ(groups string) {
	since := map[string]int{engChromium: 131, engFirefox: 132}[s.c.Engine]
	if since == 0 || s.c.Version < since {
		return
	}
	pq := strings.Contains(groups, "11ec")
	r := scoreResult{ID: "TLS-PQ", Family: famTLS, Name: "Troca de chaves pós-quântica", Value: yesNo(pq), Expected: txtYes, Weight: 5, State: stIdeal}
	if !pq {
		r.State, r.Points = stAtypical, 5
		r.Reason = s.c.Browser + " " + strconv.Itoa(since) + "+ oferece X25519MLKEM768 por padrão."
		r.Benign = "Política da empresa que desliga o pós-quântico."
	}
	s.add(&r)
}

func (s *scorer) tlsECH(ech bool) {
	if s.c.Engine != engChromium && s.c.Engine != engFirefox {
		return
	}
	r := scoreResult{ID: "TLS-ECH", Family: famTLS, Name: "Extensão ECH", Value: yesNo(ech), Expected: txtYes, Weight: 5, State: stIdeal}
	if !ech {
		r.State, r.Points = stAtypical, 5
		r.Reason = "Chrome e Firefox enviam a extensão ECH (real ou GREASE)."
		r.Benign = "Política ou versão antiga."
	}
	s.add(&r)
}

func (s *scorer) httpRules() {
	order := s.tx("Header-Order")
	var names []string
	if order != "" {
		names = strings.Split(order, ",")
	}
	secure := s.r.TLS != nil || s.tx("Tls") != "" || s.tx("Ja4") != "" || strings.EqualFold(s.r.Header.Get("X-Forwarded-Proto"), "https")

	s.httpCount(names)
	s.httpFetch(secure)
	if secure {
		s.httpHints()
		s.httpEncoding()
	}
	s.httpPlatform()
	s.httpLanguage()
	if names != nil {
		s.httpOrder(order, names)
	}
	if s.tx("Proto") == "h1" && names != nil {
		s.httpCase(names)
	}
	if s.tx("Proto") == protoH2 {
		s.httpH2()
	}

	s.matchBonus(famHTTP, "HTTP-MATCH", "Conjunto HTTP coerente", -15, 4)
	s.matchBonus(famTLS, "TLS-MATCH", "ClientHello coerente", -10, 3)
}

func (s *scorer) httpCount(names []string) {
	if names == nil {
		s.unknown("HTTP-COUNT", famHTTP, "Quantidade de cabeçalhos", 20, "Sem X-Traefix-Header-Order.")
		return
	}
	r := scoreResult{ID: "HTTP-COUNT", Family: famHTTP, Name: "Quantidade de cabeçalhos", Value: strconv.Itoa(len(names)), Expected: "10 ou mais", Weight: 20, State: stIdeal}
	switch {
	case len(names) < 8:
		r.State, r.Points = stContradiction, 20
		r.Reason = "Uma navegação de " + s.c.Browser + " tem de 10 a 14 cabeçalhos; curl manda 3, Go 3."
		r.Benign = "Proxy que remove cabeçalhos."
	case len(names) < 10:
		r.State = stNormal
	}
	s.add(&r)
}

func (s *scorer) httpFetch(secure bool) {
	h := s.r.Header
	mode, dest, site := h.Get("Sec-Fetch-Mode"), h.Get("Sec-Fetch-Dest"), h.Get("Sec-Fetch-Site")
	r := scoreResult{
		ID: "HTTP-FETCH", Family: famHTTP, Name: "Fetch Metadata (Sec-Fetch-*)", Weight: 25, State: stIdeal,
		Value: strings.Trim(mode+" "+dest+" "+site, " "), Expected: "Sec-Fetch-* enviados",
	}
	if s.nav {
		r.Expected = "navigate document"
	}
	switch {
	case !secure:
		r.State = stNormal
		r.Reason = "Sem HTTPS: navegadores só enviam Sec-Fetch-* em contexto seguro."
	case mode == "" && dest == "" && site == "":
		r.Value = txtMissing
		r.State, r.Points = stContradiction, 25
		r.Reason = "Chrome, Firefox e Safari 16.4+ enviam Sec-Fetch-* em toda requisição HTTPS."
		r.Benign = "Navegador muito antigo."
	case s.nav && (mode != "navigate" || dest != "document"):
		r.State, r.Points = stContradiction, 15
		r.Reason = "Esta página é uma navegação: o navegador manda mode=navigate e dest=document."
	}
	s.add(&r)
}

func (s *scorer) httpHints() {
	ch := s.r.Header.Get("Sec-Ch-Ua")
	r := scoreResult{ID: "HTTP-CH", Family: famHTTP, Name: "Client Hints (Sec-CH-UA)", Value: yesNo(ch != ""), Weight: 30, State: stIdeal}
	if s.c.Engine == engChromium {
		r.Expected = txtYes
		if ch == "" {
			r.State, r.Points = stContradiction, 25
			r.Reason = "Navegadores Chromium enviam Sec-CH-UA em HTTPS."
			r.Benign = "Navegador Chromium de nicho."
		}
	} else {
		r.Expected = txtNo
		if ch != "" {
			r.State, r.Points = stContradiction, 30
			r.Reason = s.c.Browser + " não implementa Client Hints: o cabeçalho foi fabricado."
		}
	}
	s.add(&r)
}

var chPlatforms = map[string]string{"Windows": osWindows, "macOS": osMacOS, "Linux": osLinux, "Android": osAndroid, "Chrome OS": osChromeOS, "iOS": osIOS}

func (s *scorer) httpPlatform() {
	if plat := strings.Trim(s.r.Header.Get("Sec-Ch-Ua-Platform"), `"`); plat != "" {
		r := scoreResult{ID: "HTTP-CH-PLAT", Family: famHTTP, Name: "Plataforma dos Client Hints", Value: plat, Expected: s.c.OS, Weight: 30, State: stIdeal}
		if os := chPlatforms[plat]; os != "" && s.c.OS != "" && os != s.c.OS {
			r.State, r.Points = stContradiction, 30
			r.Reason = "Sec-CH-UA-Platform diz " + plat + ", o User-Agent diz outro sistema."
		}
		s.add(&r)
	}

	if mob := s.r.Header.Get("Sec-Ch-Ua-Mobile"); mob == "?0" || mob == "?1" {
		r := scoreResult{ID: "HTTP-CH-MOB", Family: famHTTP, Name: "Celular nos Client Hints", Value: yesNo(mob == "?1"), Expected: yesNo(s.c.Mobile), Weight: 10, State: stIdeal}
		if (mob == "?1") != s.c.Mobile {
			r.State, r.Points = stContradiction, 10
			r.Reason = "Sec-CH-UA-Mobile difere do User-Agent."
			r.Benign = "Emulação de dispositivo do DevTools."
		}
		s.add(&r)
	}
}

func (s *scorer) httpEncoding() {
	enc := s.r.Header.Get("Accept-Encoding")
	r := scoreResult{ID: "HTTP-ENC", Family: famHTTP, Name: "Compressão aceita", Value: enc, Expected: "com br", Weight: 20, State: stIdeal}
	if !strings.Contains(enc, "br") {
		r.State, r.Points = stContradiction, 20
		r.Reason = "Navegadores aceitam Brotli (br) em HTTPS."
		r.Benign = "Proxy transparente."
		if enc == "" {
			r.Value = txtMissing
		}
	}
	s.add(&r)
}

func (s *scorer) httpLanguage() {
	lang := s.r.Header.Get("Accept-Language")
	r := scoreResult{ID: "HTTP-LANG", Family: famHTTP, Name: "Idioma (Accept-Language)", Value: lang, Expected: "presente", Weight: 15, State: stIdeal}
	if lang == "" {
		r.Value = txtMissing
		r.State, r.Points = stContradiction, 15
		r.Reason = "Navegadores sempre mandam o idioma."
		r.Benign = "resistFingerprinting do Firefox."
	}
	s.add(&r)
}

func (s *scorer) httpOrder(order string, names []string) {
	if s.c.Engine != engChromium && s.c.Engine != engFirefox {
		return
	}
	idx := func(n string) int {
		return slices.IndexFunc(names, func(x string) bool { return strings.EqualFold(x, n) })
	}
	ua, acc, enc, lg := idx("user-agent"), idx("accept"), idx("accept-encoding"), idx("accept-language")
	r := scoreResult{ID: "HTTP-ORDER", Family: famHTTP, Name: "Ordem dos cabeçalhos", Value: order, Weight: 20, State: stIdeal}
	if ua < 0 || acc < 0 || enc < 0 || lg < 0 {
		r.State = stUnknown
		r.Reason = "Faltam cabeçalhos para comparar a ordem."
		s.add(&r)
		return
	}
	ok := ua < acc && enc < lg
	r.Expected = "user-agent antes de accept; accept-encoding antes de accept-language"
	if s.c.Engine == engFirefox {
		ok = ua < acc && acc < lg && lg < enc
		r.Expected = "user-agent, accept, accept-language, accept-encoding"
	}
	if !ok {
		r.State, r.Points = stContradiction, 20
		r.Reason = "A ordem não é a do " + s.c.Browser + "."
		r.Benign = "Extensão que injeta cabeçalhos."
	}
	s.add(&r)
}

func (s *scorer) httpCase(names []string) {
	r := scoreResult{ID: "HTTP-CASE", Family: famHTTP, Name: "Maiúsculas no HTTP/1.1", Weight: 10, State: stIdeal, Expected: "User-Agent"}
	switch {
	case slices.Contains(names, "user-agent"):
		r.Value = "user-agent"
		r.State, r.Points = stContradiction, 10
		r.Reason = "Navegadores escrevem User-Agent com maiúsculas no HTTP/1.1."
		r.Benign = "Proxy HTTP/1 (não prova automação)."
	case slices.Contains(names, "User-Agent"):
		r.Value = "User-Agent"
	default:
		r.State = stUnknown
	}
	s.add(&r)
}

func (s *scorer) httpH2() {
	ref, ok := h2Refs[s.c.Engine]
	if !ok {
		return
	}
	parts := strings.Split(s.tx("H2"), "|")
	if len(parts) != 4 {
		s.unknown("HTTP-H2", famHTTP, "HTTP/2 (Akamai)", 30, "Sem X-Traefix-H2.")
		return
	}

	r := scoreResult{ID: "HTTP-H2-PSEUDO", Family: famHTTP, Name: "Ordem dos pseudo-cabeçalhos", Value: parts[3], Expected: ref.Pseudo, Weight: 30, State: stIdeal}
	if parts[3] != ref.Pseudo {
		r.State, r.Points = stContradiction, 30
		r.Reason = "Ordem de outra implementação"
		if lib := h2Libraries[parts[3]]; lib != "" {
			r.Reason += ": igual à do " + lib
		}
		r.Reason += "."
	}
	s.add(&r)

	r = scoreResult{
		ID: "HTTP-H2-SET", Family: famHTTP, Name: "SETTINGS do HTTP/2", Value: parts[0] + "|" + parts[1],
		Expected: ref.Settings + "|" + ref.Window, Weight: 20, State: stIdeal,
	}
	if parts[0] != ref.Settings || parts[1] != ref.Window {
		r.State, r.Points = stContradiction, 20
		r.Reason = "SETTINGS e WINDOW_UPDATE diferentes dos do " + s.c.Browser + " medido."
		r.Benign = "Versão do navegador que mudou os valores."
	}
	s.add(&r)
}

// matchBonus gives the relief of a family whose rules agree with the declared client, unless a known client
// already gave it.
func (s *scorer) matchBonus(family, id, name string, points, minRules int) {
	n := 0
	for _, r := range s.results {
		if r.Family != family || r.State == stUnknown {
			continue
		}
		if r.Points != 0 {
			return
		}
		n++
	}
	if n >= minRules {
		s.add(&scoreResult{
			ID: id, Family: family, Name: name, State: stIdeal, Points: points, Weight: -points,
			Reason: "Tudo coerente com " + s.c.Browser + ". Coerência não prova humano: o teto negativo é pequeno.",
		})
	}
}

func (s *scorer) tcpRules(m *netMatch) {
	t := kv(s.tx("Transport"))
	win, opts := t["syn_win"], t["syn_opts"]
	if p := strings.Split(s.tx("Ja4t"), "_"); len(p) == 4 {
		win, opts = p[0], p[1]
	}

	mismatch := false
	switch {
	case t["proto"] == "tcp" || opts != "":
		if opts == "" || opts == "2" {
			s.add(&scoreResult{
				ID: "TCP-RAW", Family: famTCP, Name: "Opções do SYN", Value: opts, Expected: "opções completas", Weight: 25,
				State: stContradiction, Points: 25, Reason: "SYN mínimo, típico de ferramenta de pacote cru (masscan, scanners).",
			})
		}
		mismatch = s.tcpOS(win, opts, m)
	case s.browser && t["proto"] != "udp":
		s.unknown("TCP-OS", famTCP, "Sistema pela pilha TCP", 20, "Sem o SYN (opção tcp desligada ou HTTP/3).")
	}

	s.tcpTTL(t["ttl_init"], mismatch, m)

	if hops, ok := atoi(t["hops"]); ok && s.browser && hops <= 2 {
		s.add(&scoreResult{
			ID: "TCP-HOPS", Family: famTCP, Name: "Saltos de rede", Value: strconv.Itoa(hops), Expected: "mais de 2", Weight: 5,
			State: stAtypical, Points: 5, Reason: "Origem na mesma rede do servidor.", Benign: "Rede local, teste do próprio dono.",
		})
	}
}

// tcpOS compares the stack of the SYN with the system of the User-Agent, and tells whether they differ.
func (s *scorer) tcpOS(win, opts string, m *netMatch) bool {
	want := osStack(s.c.OS)
	if !s.browser || want == "" {
		return false
	}
	syn := synStack(win, opts)
	r := scoreResult{ID: "TCP-OS", Family: famTCP, Name: "Sistema pela pilha TCP", Value: stackNames[syn], Expected: stackNames[want], Weight: 20}
	mismatch := false
	switch {
	case opts == "":
		r.State, r.Reason = stUnknown, "Sem o SYN (opção tcp desligada)."
	case syn == "":
		r.State, r.Value, r.Reason = stNormal, opts, "Pilha TCP não catalogada."
	case syn == want:
		r.State, r.Points = stIdeal, -10
		r.Reason = "O SYN é de " + stackNames[syn] + ", como o User-Agent diz."
	case m.Relay != "":
		r.State, r.Reason = stNormal, "O SYN é do "+m.Relay+", não do seu aparelho."
	default:
		mismatch = true
		r.State, r.Points = stContradiction, 20
		r.Reason = "O SYN é de " + stackNames[syn] + ", o User-Agent diz " + stackNames[want] + "."
		r.Benign = "VPN, proxy ou relay que termina o TCP; máquina virtual."
	}
	s.add(&r)
	return mismatch
}

func (s *scorer) tcpTTL(ttlInit string, mismatch bool, m *netMatch) {
	stack := osStack(s.c.OS)
	if ttlInit == "" || !s.browser || stack == "" {
		return
	}
	want := "64"
	if stack == osWindows {
		want = "128"
	}
	r := scoreResult{ID: "TCP-TTL", Family: famTCP, Name: "TTL inicial", Value: ttlInit, Expected: want, Weight: 10, State: stIdeal}
	switch {
	case ttlInit == want:
	case m.Relay != "":
		r.State, r.Reason = stNormal, "TTL do "+m.Relay+"."
	case mismatch:
		r.State, r.Reason = stNormal, "Mesma causa do sistema pela pilha TCP: não conta duas vezes."
	default:
		r.State, r.Points = stContradiction, 10
		r.Reason = "TTL inicial de outro sistema que o do User-Agent."
		r.Benign = "VPN, proxy ou roteador que reescreve o TTL."
	}
	s.add(&r)
}

// Thresholds of TIME-RELAY, from the measures of finger-sniffer on finger-01 (task 313, SOCKS5 relay and direct
// clients): rtt_gap > 10 ms with est_to_first > 5 ms caught 43 of 50 relayed connections and 2 of 40 direct ones.
const (
	relayGapMicros   = 10_000
	relayGapFrac     = 0.3
	relayFirstMicros = 5_000
)

// timeRules looks for a proxy or SOCKS between the client and the Traefix. The TCP handshake ends at whoever opened
// the TCP connection, the TLS round trip goes up to the browser: through a relay, the TLS round trip exceeds the
// TCP one by the distance from the relay to the client (rtt_gap_us), and the first byte of the ClientHello comes one
// relay round trip after the ACK (est_to_first_us), because the relay only tells the client that the tunnel is up
// after it. A direct client sends the ClientHello right after the ACK. It reports whether the relay is confirmed.
func (s *scorer) timeRules(m *netMatch) bool {
	t := kv(s.tx("Timing"))
	gap, ok1 := atoi(t["rtt_gap_us"])
	rtt, ok2 := atoi(t["tcp_rtt_us"])
	if !ok1 || !ok2 {
		if s.browser && s.tx("Proto") != "h3" {
			s.unknown(idTimeRelay, famTime, "Proxy ou SOCKS pelo tempo", 60, "Sem X-Traefix-Timing (headerDetail basic).")
		}
		return false
	}
	est, okEst := atoi(t["est_to_first_us"])
	value := "TLS +" + msText(gap) + " sobre o TCP"
	if okEst {
		value += "; ClientHello " + msText(est) + " após o ACK"
	}
	r := scoreResult{
		ID: idTimeRelay, Family: famTime, Name: "Proxy ou SOCKS pelo tempo", Value: value,
		Expected: "diferença perto de 0 e ClientHello em até 5 ms", Weight: 60, State: stIdeal,
	}
	slowTLS := gap > relayGapMicros && float64(gap) > relayGapFrac*float64(rtt)
	confirmed := false
	switch {
	case !slowTLS:
	case m.Relay != "":
		r.State, r.Reason = stNormal, "Atraso do "+m.Relay+", relay de privacidade da Apple: não conta."
	case okEst && est > relayFirstMicros:
		r.State, r.Points, confirmed = stContradiction, 60, true
		r.Reason = "O TLS levou " + msText(gap) + " a mais que o TCP e o ClientHello chegou " + msText(est) +
			" depois do ACK: quem abriu o TCP não é o navegador. Proxy ou SOCKS no caminho, recurso comum para contornar limites por IP."
		r.Benign = "VPN corporativa ou proxy de empresa; Wi-Fi com pico de atraso na mesma conexão (raro com os dois sinais juntos)."
	default:
		r.State, r.Points = stAtypical, 5
		r.Reason = "O TLS demorou, mas o ClientHello chegou logo após o ACK: mais provável um dispositivo lento ou Wi-Fi que um proxy."
	}
	s.add(&r)
	return confirmed
}

// noRelief zeroes the negative points, appending why to the reason.
func (s *scorer) noRelief(why string) {
	for i := range s.results {
		if s.results[i].Points < 0 {
			s.results[i].Points = 0
			s.results[i].Reason += why
		}
	}
}

// msText formats microseconds as milliseconds with one decimal.
func msText(us int) string {
	return strconv.FormatFloat(float64(us)/1000, 'f', 1, 64) + " ms"
}

func (s *scorer) netRules(store *listStore, m *netMatch, role string) {
	if store == nil {
		return
	}
	if !m.Loaded {
		s.unknown("NET-CLOUD", famNet, "Rede de datacenter", 20, "As listas de endereços ainda não foram baixadas.")
		return
	}
	if m.Relay != "" {
		v := m.Relay
		if m.RelayDetail != "" {
			v += " (" + m.RelayDetail + ")"
		}
		s.add(&scoreResult{
			ID: "NET-RELAY", Family: famNet, Name: "Relay de privacidade", Value: v, State: stNormal,
			Reason: "Saída do iCloud Private Relay: TCP, TTL e tempos são do relay, e não contam.",
		})
	}

	r := scoreResult{ID: "NET-CLOUD", Family: famNet, Name: "Rede de datacenter", Value: m.Cloud, Expected: "rede de acesso", Weight: 20, State: stIdeal}
	switch {
	case m.Cloud == "":
		r.Value = txtNo
	case m.Relay != "" || role == roleCrawlerVerified:
		r.State = stNormal
	case s.browser:
		r.State, r.Points = stAtypical, 20
		r.Reason = "Endereço da " + m.Cloud + " com UA de navegador: pessoas raramente navegam de um datacenter."
		r.Benign = "VPN, VDI, proxy de empresa na nuvem."
	default:
		r.State = stNormal
		r.Reason = "Programa rodando em " + m.Cloud + "."
	}
	s.add(&r)

	if len(m.Crawlers) > 0 && s.c.Crawler == "" {
		r = scoreResult{
			ID: "NET-CRAWLER-IP", Family: famNet, Name: "Endereço de robô", Value: m.CrawlerLabel + " (" + strings.Join(m.Crawlers, ", ") + ")",
			Weight: 10, State: stNormal, Reason: "O endereço está numa lista oficial de robôs.",
		}
		if s.browser {
			r.State, r.Points = stAtypical, 10
			r.Reason += " Com UA de navegador: buscador sob demanda do operador (pré-visualização, verificação)."
		}
		s.add(&r)
	}
}

// aggregate caps the points per family and maps the sum through the sigmoid of SCORE.md.
func aggregate(results []scoreResult) scoreSummary {
	sum := make(map[string]int)
	evaluated, total := 0, 0
	for _, r := range results {
		sum[r.Family] += r.Points
		w := r.Weight
		if w < 0 {
			w = -w
		}
		total += w
		if r.State != stUnknown {
			evaluated += w
		}
	}

	out := scoreSummary{Families: make(map[string]int)}
	for f, v := range sum {
		c := familyCaps[f]
		v = min(max(v, c.Min), c.Max)
		out.Families[f] = v
		out.S += v
	}
	out.R = int(math.Round(100 / (1 + math.Exp(-(scoreA*float64(out.S) + scoreB)))))
	if total > 0 {
		out.Coverage = int(math.Round(100 * float64(evaluated) / float64(total)))
	}
	out.Insufficient = out.Coverage < 50
	out.Band = band(out.R, out.Insufficient)
	return out
}

func band(r int, insufficient bool) string {
	switch {
	case insufficient:
		return "Evidência insuficiente"
	case r <= 20:
		return "Compatível com navegador"
	case r <= 50:
		return "Indeterminado"
	case r <= 80:
		return "Suspeito de automação"
	}
	return "Automação provável"
}

// writeScoreText adds the passive score to the text output, for curl and scripts.
func writeScoreText(w io.Writer, sc *Score) {
	p := sc.Passive
	_, _ = fmt.Fprintf(w, "Automation-Score: %d/100 (%s, cobertura %d%%, %s, índice não calibrado)\n", p.R, p.Band, p.Coverage, sc.Role)
	if sc.Client.Known != "" {
		_, _ = fmt.Fprintln(w, "Automation-Known-TLS:", sc.Client.Known)
	}
	for _, r := range sc.Results {
		if r.Points == 0 {
			continue
		}
		_, _ = fmt.Fprintf(w, "Automation-Rule: %+d %s %s\n", r.Points, r.ID, r.Reason)
	}
}
