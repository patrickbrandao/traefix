package headers

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/middlewares"
	"github.com/vulcand/oxy/v2/forward"
)

// Header is a middleware that helps setup a few basic security features.
// A single headerOptions struct can be provided to configure which features should be enabled,
// and the ability to override a few of the default values.
type Header struct {
	next               http.Handler
	hasCustomHeaders   bool
	hasCorsHeaders     bool
	headers            *dynamic.Headers
	allowOriginRegexes []*regexp.Regexp
	replaceReqRegexes  map[string]headerReplacement
	replaceRespRegexes map[string]headerReplacement
}

type headerReplacement struct {
	regexp      *regexp.Regexp
	replacement string
}

// NewHeader constructs a new header instance from supplied frontend header struct.
func NewHeader(next http.Handler, cfg dynamic.Headers) (*Header, error) {
	hasCustomHeaders := cfg.HasCustomHeadersDefined()
	hasCorsHeaders := cfg.HasCorsHeadersDefined()

	regexes := make([]*regexp.Regexp, len(cfg.AccessControlAllowOriginListRegex))
	for i, str := range cfg.AccessControlAllowOriginListRegex {
		reg, err := regexp.Compile(str)
		if err != nil {
			return nil, fmt.Errorf("error occurred during origin parsing: %w", err)
		}
		regexes[i] = reg
	}

	replaceReqRegexes, err := compileHeaderReplacements(cfg.ReplaceRequestHeadersRegex)
	if err != nil {
		return nil, fmt.Errorf("replaceRequestHeadersRegex: %w", err)
	}

	replaceRespRegexes, err := compileHeaderReplacements(cfg.ReplaceResponseHeadersRegex)
	if err != nil {
		return nil, fmt.Errorf("replaceResponseHeadersRegex: %w", err)
	}

	return &Header{
		next:               next,
		headers:            &cfg,
		hasCustomHeaders:   hasCustomHeaders,
		hasCorsHeaders:     hasCorsHeaders,
		allowOriginRegexes: regexes,
		replaceReqRegexes:  replaceReqRegexes,
		replaceRespRegexes: replaceRespRegexes,
	}, nil
}

func compileHeaderReplacements(cfg map[string]dynamic.ReplaceHeaderRegex) (map[string]headerReplacement, error) {
	if len(cfg) == 0 {
		return nil, nil
	}

	replacements := make(map[string]headerReplacement, len(cfg))
	for header, rep := range cfg {
		// An empty regex matches between every character and would mangle the whole value.
		if rep.Regex == "" {
			return nil, fmt.Errorf("empty regex for header %q", header)
		}

		reg, err := regexp.Compile(rep.Regex)
		if err != nil {
			return nil, fmt.Errorf("parsing regex for header %q: %w", header, err)
		}

		replacements[http.CanonicalHeaderKey(header)] = headerReplacement{
			regexp:      reg,
			replacement: rep.Replacement,
		}
	}

	return replacements, nil
}

// replaceHeaderValues applies the replacements to every value of the headers, so that multi-valued headers such as Set-Cookie are kept.
// A value replaced by an empty string is removed.
func replaceHeaderValues(h http.Header, replacements map[string]headerReplacement) {
	for header, rep := range replacements {
		values := h.Values(header)
		if len(values) == 0 {
			continue
		}

		newValues := make([]string, 0, len(values))
		for _, value := range values {
			if value = rep.regexp.ReplaceAllString(value, rep.replacement); value != "" {
				newValues = append(newValues, value)
			}
		}

		if len(newValues) == 0 {
			h.Del(header)
			continue
		}

		h[header] = newValues
	}
}

func (s *Header) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Handle Cors headers and preflight if configured.
	if isPreflight := s.processCorsHeaders(rw, req); isPreflight {
		rw.Header().Set("Content-Length", "0")
		rw.WriteHeader(http.StatusOK)
		return
	}

	if s.hasCustomHeaders {
		s.modifyCustomRequestHeaders(req)
	}

	// If there is a next, call it.
	if s.next != nil {
		s.next.ServeHTTP(middlewares.NewResponseModifier(rw, req, s.PostRequestModifyResponseHeaders), req)
	}
}

// PostRequestModifyResponseHeaders set or delete response headers.
// This method is called AFTER the response is generated from the backend
// and can merge/override headers from the backend response.
func (s *Header) PostRequestModifyResponseHeaders(res *http.Response) error {
	// Loop through Custom response headers
	for header, value := range s.headers.CustomResponseHeaders {
		if value == "" {
			res.Header.Del(header)
		} else {
			res.Header.Set(header, value)
		}
	}

	replaceHeaderValues(res.Header, s.replaceRespRegexes)

	if res != nil && res.Request != nil {
		match, ok := s.matchOrigin(res.Request.Header.Get("Origin"))
		if ok {
			res.Header.Set("Access-Control-Allow-Origin", match)

			// Access-Control-* headers are only meaningful alongside a matching Access-Control-Allow-Origin:
			// per the W3C CORS Recommendation's §7.2 "Resource Sharing Check"
			// (https://www.w3.org/TR/2020/SPSD-cors-20200602/#resource-requests), and its modern equivalent, the
			// "CORS check" in the WHATWG Fetch Standard (https://fetch.spec.whatwg.org/#cors-check).
			// The request gets a network error as soon as the Origin fails to match.
			if s.headers.AccessControlAllowCredentials {
				res.Header.Set("Access-Control-Allow-Credentials", "true")
			}

			if len(s.headers.AccessControlExposeHeaders) > 0 {
				exposeHeaders := strings.Join(s.headers.AccessControlExposeHeaders, ",")
				if !(slices.Contains(s.headers.AccessControlExposeHeaders, "*") && s.headers.AccessControlAllowCredentials) {
					res.Header.Set("Access-Control-Expose-Headers", exposeHeaders)
				}
			}
		}
	}

	if !s.headers.AddVaryHeader {
		return nil
	}

	// The backend may send Vary on several lines or as a list on one line,
	// and every value must be kept so that caches do not serve the wrong variant.
	varyValues := res.Header.Values("Vary")
	for _, value := range varyValues {
		for token := range strings.SplitSeq(value, ",") {
			token = strings.TrimSpace(token)
			if token == "*" || strings.EqualFold(token, "Origin") {
				return nil
			}
		}
	}

	res.Header.Set("Vary", strings.Join(append(slices.Clone(varyValues), "Origin"), ","))
	return nil
}

// modifyCustomRequestHeaders sets or deletes custom request headers.
func (s *Header) modifyCustomRequestHeaders(req *http.Request) {
	// Note: the header names aliasing the custom request headers (e.g. X_Auth_User) are not handled here,
	// as the aliasHeadersStrategy entry point option is expected to be enabled to prevent header spoofing.
	// Loop through Custom request headers
	for header, value := range s.headers.CustomRequestHeaders {
		switch {
		// Handling https://github.com/golang/go/commit/ecdbffd4ec68b509998792f120868fec319de59b.
		case value == "" && header == forward.XForwardedFor:
			req.Header[header] = nil

		case value == "":
			req.Header.Del(header)

		case strings.EqualFold(header, "Host"):
			req.Host = value

		default:
			req.Header.Set(header, value)
		}
	}

	replaceHeaderValues(req.Header, s.replaceReqRegexes)
}

// processCorsHeaders processes the incoming request,
// and returns if it is a preflight request.
// If not a preflight, it handles the preRequestModifyCorsResponseHeaders.
func (s *Header) processCorsHeaders(rw http.ResponseWriter, req *http.Request) bool {
	if !s.hasCorsHeaders {
		return false
	}

	reqAcMethod := req.Header.Get("Access-Control-Request-Method")
	originHeader := req.Header.Get("Origin")

	if reqAcMethod != "" && originHeader != "" && req.Method == http.MethodOptions {
		// If the request is an OPTIONS request with an Access-Control-Request-Method header,
		// and Origin headers, then it is a CORS preflight request,
		// and we need to build a custom response: https://www.w3.org/TR/cors/#preflight-request

		// Per the Fetch Standard's "CORS-preflight fetch" algorithm (https://fetch.spec.whatwg.org/#cors-preflight-fetch),
		// and formerly the W3C CORS Recommendation's equivalent §7.1.5 "Cross-Origin Request with Preflight"
		// (https://www.w3.org/TR/2020/SPSD-cors-20200602/#resource-preflight-requests), the preflight response is validated
		// against the Origin via the same Resource Sharing Check used for actual requests before other Access-Control-* headers
		// are inspected, and a mismatch discards the whole preflight response as a network error.
		match, ok := s.matchOrigin(originHeader)
		if !ok {
			return true
		}

		rw.Header().Set("Access-Control-Allow-Origin", match)

		if s.headers.AccessControlAllowCredentials {
			rw.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		allowHeaders := strings.Join(s.headers.AccessControlAllowHeaders, ",")
		if slices.Contains(s.headers.AccessControlAllowHeaders, "*") && s.headers.AccessControlAllowCredentials {
			if acrh := req.Header.Get("Access-Control-Request-Headers"); acrh != "" {
				rw.Header().Set("Access-Control-Allow-Headers", acrh)
			}
		} else if allowHeaders != "" {
			rw.Header().Set("Access-Control-Allow-Headers", allowHeaders)
		}

		allowMethods := strings.Join(s.headers.AccessControlAllowMethods, ",")
		if slices.Contains(s.headers.AccessControlAllowMethods, "*") && s.headers.AccessControlAllowCredentials {
			rw.Header().Set("Access-Control-Allow-Methods", reqAcMethod)
		} else if allowMethods != "" {
			rw.Header().Set("Access-Control-Allow-Methods", allowMethods)
		}

		if s.headers.AccessControlMaxAge != nil {
			rw.Header().Set("Access-Control-Max-Age", strconv.FormatInt(*s.headers.AccessControlMaxAge, 10))
		}

		if len(s.headers.AccessControlExposeHeaders) > 0 {
			exposeHeaders := strings.Join(s.headers.AccessControlExposeHeaders, ",")
			if !(slices.Contains(s.headers.AccessControlExposeHeaders, "*") && s.headers.AccessControlAllowCredentials) {
				rw.Header().Set("Access-Control-Expose-Headers", exposeHeaders)
			}
		}

		return true
	}

	return false
}

func (s *Header) matchOrigin(origin string) (string, bool) {
	for _, item := range s.headers.AccessControlAllowOriginList {
		switch item {
		case origin:
			return item, true
		case "*":
			if s.headers.AccessControlAllowCredentials {
				return origin, true // Can't use wildcard with credentials.
			}
			return item, true
		}
	}

	for _, regex := range s.allowOriginRegexes {
		if regex.MatchString(origin) {
			return origin, true
		}
	}

	return "", false
}
