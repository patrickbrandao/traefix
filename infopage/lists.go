package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"
)

// listKind tells what an address list means for the score.
type listKind string

const (
	kindCrawler listKind = "crawler"
	kindCloud   listKind = "cloud"
	kindRelay   listKind = "relay"
)

// ipList is a published list of address ranges. The crawler lists are the ones the operators publish to verify
// their robots; the cloud lists come from the providers; the iCloud Private Relay list is an exception, people
// behind a relay, so that its TCP stack and its round trips aren't held against a Safari user.
type ipList struct {
	Name  string
	Label string
	URL   string
	Kind  listKind
	Robot string
	CSV   bool
}

const (
	labelGoogle = "Google"
	labelOpenAI = "OpenAI"
)

var ipLists = []ipList{
	{Name: "googlebot", Label: labelGoogle, URL: "https://developers.google.com/static/search/apis/ipranges/googlebot.json", Kind: kindCrawler, Robot: robotGoogle},
	{Name: "google_special", Label: labelGoogle, URL: "https://developers.google.com/static/search/apis/ipranges/special-crawlers.json", Kind: kindCrawler, Robot: robotGoogle},
	{Name: "google_user", Label: labelGoogle, URL: "https://developers.google.com/static/search/apis/ipranges/user-triggered-fetchers.json", Kind: kindCrawler, Robot: robotGoogle},
	{Name: "bingbot", Label: "Microsoft Bing", URL: "https://www.bing.com/toolbox/bingbot.json", Kind: kindCrawler, Robot: robotBing},
	{Name: "applebot", Label: "Apple", URL: "https://search.developer.apple.com/applebot.json", Kind: kindCrawler, Robot: robotApple},
	{Name: "gptbot", Label: labelOpenAI, URL: "https://openai.com/gptbot.json", Kind: kindCrawler, Robot: robotGPT},
	{Name: "oai_searchbot", Label: labelOpenAI, URL: "https://openai.com/searchbot.json", Kind: kindCrawler, Robot: robotOAISearch},
	{Name: "chatgpt_user", Label: labelOpenAI, URL: "https://openai.com/chatgpt-user.json", Kind: kindCrawler, Robot: robotChatGPT},
	{Name: "duckduckbot", Label: "DuckDuckGo", URL: "https://duckduckgo.com/duckduckbot.json", Kind: kindCrawler, Robot: robotDuckDuckGo},
	{Name: "perplexitybot", Label: "Perplexity", URL: "https://www.perplexity.com/perplexitybot.json", Kind: kindCrawler, Robot: robotPerplexity},
	{Name: "perplexity_user", Label: "Perplexity", URL: "https://www.perplexity.com/perplexity-user.json", Kind: kindCrawler, Robot: robotPerplexityUsr},
	{Name: "ccbot", Label: "Common Crawl", URL: "https://index.commoncrawl.org/ccbot.json", Kind: kindCrawler, Robot: robotCommonCrawl},
	{Name: "aws", Label: "Amazon AWS", URL: "https://ip-ranges.amazonaws.com/ip-ranges.json", Kind: kindCloud},
	{Name: "gcp", Label: "Google Cloud", URL: "https://www.gstatic.com/ipranges/cloud.json", Kind: kindCloud},
	{Name: "oracle", Label: "Oracle Cloud", URL: "https://docs.oracle.com/en-us/iaas/tools/public_ip_ranges.json", Kind: kindCloud},
	{Name: "digitalocean", Label: "DigitalOcean", URL: "https://digitalocean.com/geo/google.csv", Kind: kindCloud, CSV: true},
	{Name: "cloudflare", Label: "Cloudflare", URL: "https://api.cloudflare.com/client/v4/ips", Kind: kindCloud},
	{Name: "icloud_relay", Label: "iCloud Private Relay", URL: "https://mask-api.icloud.com/egress-ip-ranges.csv", Kind: kindRelay, CSV: true},
}

// robotSuffixes are the reverse DNS suffixes the operators publish; a robot without suffix is verified by its
// list alone.
var robotSuffixes = map[string][]string{
	robotGoogle: {"googlebot.com", "google.com", "googleusercontent.com"},
	robotBing:   {"search.msn.com"},
	robotApple:  {"applebot.apple.com"},
}

const (
	listMaxBytes = 64 << 20
	listRefresh  = 24 * time.Hour
	listRetry    = time.Hour
)

// ipRange holds the addresses in their 16-byte form (IPv4 mapped), without the pointer of netip.Addr: the
// Private Relay list alone has 285 thousand ranges, and the container has 64 MB.
type ipRange struct {
	lo, hi [16]byte
	// detail indexes rangeSet.details, the location of a Private Relay egress ("BR,BR-SP,São Paulo").
	detail uint32
}

type rangeSet struct {
	ranges  []ipRange
	details []string
	fetched time.Time
	count   int
}

func (s *rangeSet) lookup(ip netip.Addr) (string, bool) {
	a := ip.As16()
	i, _ := slices.BinarySearchFunc(s.ranges, a, func(r ipRange, a [16]byte) int {
		if bytes.Compare(r.hi[:], a[:]) < 0 {
			return -1
		}
		if bytes.Compare(r.lo[:], a[:]) > 0 {
			return 1
		}
		return 0
	})
	if i < len(s.ranges) && bytes.Compare(s.ranges[i].lo[:], a[:]) <= 0 && bytes.Compare(s.ranges[i].hi[:], a[:]) >= 0 {
		return s.details[s.ranges[i].detail], true
	}
	return "", false
}

// listStore keeps the last good copy of every list: a failed or suspicious download never replaces it.
type listStore struct {
	mu     sync.RWMutex
	sets   map[string]*rangeSet
	client *http.Client

	dnsMu    sync.Mutex
	dnsCache map[string]dnsEntry
	resolver *net.Resolver
}

type dnsEntry struct {
	host    string
	ok      bool
	expires time.Time
}

func newListStore() *listStore {
	return &listStore{
		sets:     make(map[string]*rangeSet),
		client:   &http.Client{Timeout: 60 * time.Second},
		dnsCache: make(map[string]dnsEntry),
		resolver: net.DefaultResolver,
	}
}

// lists is nil when INFOPAGE_LISTS is off: the score then has no network family.
var lists *listStore

func (s *listStore) run(ctx context.Context) {
	for {
		failed := s.refresh(ctx)
		wait := listRefresh
		if failed > 0 {
			wait = listRetry
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

func (s *listStore) refresh(ctx context.Context) int {
	failed := 0
	for _, l := range ipLists {
		set, err := s.fetch(ctx, l)
		if err != nil {
			failed++
			log.Printf("list %s: %v", l.Name, err)
			continue
		}

		s.mu.Lock()
		old := s.sets[l.Name]
		// A list that shrank by more than half is more likely a broken download than a real change.
		if old != nil && set.count < old.count/2 {
			s.mu.Unlock()
			failed++
			log.Printf("list %s: %d ranges against %d, keeping the previous copy", l.Name, set.count, old.count)
			continue
		}
		s.sets[l.Name] = set
		s.mu.Unlock()

		// A list is parsed once a day: give the memory of the download and of the previous copy back to the
		// system now, rather than leaving it to the scavenger, as the container has 64 MB.
		debug.FreeOSMemory()
	}
	return failed
}

func (s *listStore) fetch(ctx context.Context, l ipList) (*rangeSet, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.URL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "traefix-infopage")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	set, err := parseList(io.LimitReader(resp.Body, listMaxBytes), l.CSV)
	if err != nil {
		return nil, err
	}
	if set.count == 0 {
		return nil, errors.New("no address range")
	}
	set.fetched = time.Now()
	return set, nil
}

// parseList reads the first column of a CSV, or every string of a JSON document that is an address or a prefix,
// which covers the formats of all the lists (Google, AWS, Oracle, Cloudflare) with one parser. Both are read as
// a stream, so that a large list never sits whole in memory.
func parseList(r io.Reader, csv bool) (*rangeSet, error) {
	set := &rangeSet{details: []string{""}}
	index := map[string]uint32{"": 0}
	add := func(s, detail string) {
		p, ok := parsePrefix(s)
		if !ok {
			return
		}
		d, ok := index[detail]
		if !ok {
			d = uint32(len(set.details))
			index[detail] = d
			set.details = append(set.details, detail)
		}
		set.ranges = append(set.ranges, ipRange{lo: p.Masked().Addr().As16(), hi: lastAddr(p).As16(), detail: d})
	}

	var err error
	if csv {
		err = scanCSV(r, add)
	} else {
		err = scanJSON(r, add)
	}
	if err != nil {
		return nil, err
	}

	set.ranges = mergeRanges(set.ranges)
	set.ranges = slices.Clip(set.ranges)
	set.count = len(set.ranges)
	return set, nil
}

func scanCSV(r io.Reader, add func(s, detail string)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		first, rest, _ := strings.Cut(strings.TrimSpace(sc.Text()), ",")
		add(first, strings.TrimRight(rest, ","))
	}
	return sc.Err()
}

func scanJSON(r io.Reader, add func(s, detail string)) error {
	dec := json.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if s, ok := tok.(string); ok {
			add(s, "")
		}
	}
}

// mergeRanges sorts the ranges and merges the overlapping ones, so that the binary search finds the one that
// holds an address.
func mergeRanges(ranges []ipRange) []ipRange {
	slices.SortFunc(ranges, func(a, b ipRange) int { return bytes.Compare(a.lo[:], b.lo[:]) })
	merged := ranges[:0]
	for _, r := range ranges {
		if n := len(merged); n > 0 && bytes.Compare(merged[n-1].hi[:], r.lo[:]) >= 0 {
			if bytes.Compare(r.hi[:], merged[n-1].hi[:]) > 0 {
				merged[n-1].hi = r.hi
			}
			continue
		}
		merged = append(merged, r)
	}
	return merged
}

func parsePrefix(s string) (netip.Prefix, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Prefix{}, false
	}
	if p, err := netip.ParsePrefix(s); err == nil {
		return p, true
	}
	if a, err := netip.ParseAddr(s); err == nil {
		return netip.PrefixFrom(a.Unmap(), a.Unmap().BitLen()), true
	}
	return netip.Prefix{}, false
}

func lastAddr(p netip.Prefix) netip.Addr {
	a := p.Masked().Addr()
	b := a.AsSlice()
	for i := p.Bits(); i < len(b)*8; i++ {
		b[i/8] |= 1 << (7 - uint(i%8))
	}
	last, _ := netip.AddrFromSlice(b)
	return last
}

// netMatch is what the lists say of an address.
type netMatch struct {
	Cloud       string `json:"cloud,omitempty"`
	Relay       string `json:"relay,omitempty"`
	RelayDetail string `json:"relayDetail,omitempty"`
	// Crawlers holds the robots whose published list contains the address.
	Crawlers []string `json:"crawlers,omitempty"`
	// CrawlerLabel is the operator of the first of them.
	CrawlerLabel string `json:"crawlerLabel,omitempty"`
	// Loaded is false while no list has been downloaded yet.
	Loaded bool `json:"loaded"`
}

func (s *listStore) match(ip netip.Addr) netMatch {
	var m netMatch
	if s == nil || !ip.IsValid() {
		return m
	}
	ip = ip.Unmap()

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, l := range ipLists {
		set := s.sets[l.Name]
		if set == nil {
			continue
		}
		m.Loaded = true
		detail, ok := set.lookup(ip)
		if !ok {
			continue
		}
		switch l.Kind {
		case kindCloud:
			if m.Cloud == "" {
				m.Cloud = l.Label
			}
		case kindRelay:
			m.Relay = l.Label
			m.RelayDetail = detail
		case kindCrawler:
			if !slices.Contains(m.Crawlers, l.Robot) {
				m.Crawlers = append(m.Crawlers, l.Robot)
			}
			if m.CrawlerLabel == "" {
				m.CrawlerLabel = l.Label
			}
		}
	}
	return m
}

// listLoaded tells whether the lists of a robot are available, so that "not in the list" means something.
func (s *listStore) listLoaded(robot string) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, l := range ipLists {
		if l.Robot == robot && s.sets[l.Name] != nil {
			return true
		}
	}
	return false
}

// verifyDNS checks the reverse name of the address against the suffixes of the robot, then that the name
// resolves back to the address (FCrDNS), the method the operators publish. The answer is cached for a day.
func (s *listStore) verifyDNS(ctx context.Context, robot string, ip netip.Addr) (string, bool) {
	suffixes := robotSuffixes[robot]
	if s == nil || len(suffixes) == 0 || !ip.IsValid() {
		return "", false
	}

	key := robot + "|" + ip.String()
	s.dnsMu.Lock()
	if e, ok := s.dnsCache[key]; ok && time.Now().Before(e.expires) {
		s.dnsMu.Unlock()
		return e.host, e.ok
	}
	s.dnsMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	host, ok := "", false
	names, _ := s.resolver.LookupAddr(ctx, ip.String())
	for _, n := range names {
		h := strings.ToLower(strings.TrimSuffix(n, "."))
		if !hasDomainSuffix(h, suffixes) {
			continue
		}
		host = h
		addrs, _ := s.resolver.LookupNetIP(ctx, "ip", h)
		for _, a := range addrs {
			if a.Unmap() == ip.Unmap() {
				ok = true
				break
			}
		}
		if ok {
			break
		}
	}

	s.dnsMu.Lock()
	if len(s.dnsCache) > 10000 {
		s.dnsCache = make(map[string]dnsEntry)
	}
	s.dnsCache[key] = dnsEntry{host: host, ok: ok, expires: time.Now().Add(24 * time.Hour)}
	s.dnsMu.Unlock()
	return host, ok
}

// hasDomainSuffix matches whole labels: "evilgooglebot.com" isn't under "googlebot.com".
func hasDomainSuffix(host string, suffixes []string) bool {
	for _, s := range suffixes {
		if host == s || strings.HasSuffix(host, "."+s) {
			return true
		}
	}
	return false
}
