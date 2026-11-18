package dynamic

// +k8s:deepcopy-gen=true

// ForwardAuthFailOpen lets the requests reach the backend when the authentication server cannot give a verdict (Traefix).
// Without it, an authentication server error or timeout fails the request, as in Traefik.
type ForwardAuthFailOpen struct {
	// DenyRequiresResponseHeader is the header a 4xx response of the authentication server must carry to deny the request.
	// A 4xx response without it, e.g. for an invalid key or a rate limit of the authentication server, lets the request
	// through as a failure. When unset, every 4xx response denies the request.
	DenyRequiresResponseHeader string `json:"denyRequiresResponseHeader,omitempty" toml:"denyRequiresResponseHeader,omitempty" yaml:"denyRequiresResponseHeader,omitempty" export:"true"`
	// Header is the header set on the request let through after a failure, so that the backend knows no verdict was given.
	Header *ForwardAuthFailOpenHeader `json:"header,omitempty" toml:"header,omitempty" yaml:"header,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// ForwardAuthFailOpenHeader is the header set on a request let through after an authentication server failure.
type ForwardAuthFailOpenHeader struct {
	// Name of the header.
	Name string `json:"name,omitempty" toml:"name,omitempty" yaml:"name,omitempty" export:"true"`
	// Value of the header.
	Value string `json:"value,omitempty" toml:"value,omitempty" yaml:"value,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// FingerprintList holds the fingerprintList middleware configuration (Traefix).
// This middleware denies or allows the requests by the fingerprints the entry point computed.
type FingerprintList struct {
	// Mode defines how the lists apply: deny rejects the requests with a listed fingerprint,
	// allow rejects the requests without one.
	Mode string `json:"mode,omitempty" toml:"mode,omitempty" yaml:"mode,omitempty" export:"true"`
	// JA3 lists JA3 hashes. A trailing * matches the fingerprints starting with the given prefix.
	JA3 []string `json:"ja3,omitempty" toml:"ja3,omitempty" yaml:"ja3,omitempty" export:"true"`
	// JA4 lists JA4 fingerprints. A trailing * matches the fingerprints starting with the given prefix.
	JA4 []string `json:"ja4,omitempty" toml:"ja4,omitempty" yaml:"ja4,omitempty" export:"true"`
	// JA4H lists JA4H fingerprints. A trailing * matches the fingerprints starting with the given prefix.
	JA4H []string `json:"ja4h,omitempty" toml:"ja4h,omitempty" yaml:"ja4h,omitempty" export:"true"`
	// HTTP2 lists Akamai HTTP/2 fingerprints. A trailing * matches the fingerprints starting with the given prefix.
	HTTP2 []string `json:"http2,omitempty" toml:"http2,omitempty" yaml:"http2,omitempty" export:"true"`
	// JA4T lists JA4T fingerprints of the TCP SYN, which need the tcp option of the entry point fingerprint.
	// A trailing * matches the fingerprints starting with the given prefix.
	JA4T []string `json:"ja4t,omitempty" toml:"ja4t,omitempty" yaml:"ja4t,omitempty" export:"true"`
	// RejectStatusCode defines the HTTP status code used for refused requests.
	// If not set, the default is 403 (Forbidden).
	RejectStatusCode int `json:"rejectStatusCode,omitempty" toml:"rejectStatusCode,omitempty" yaml:"rejectStatusCode,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// Coraza holds the native OWASP Coraza WAF middleware configuration (Traefix).
// It takes the same options as the Coraza middleware of Traefik Hub, and plugin.coraza is read as this middleware
// when no plugin of that name is declared.
type Coraza struct {
	// Directives lists the SecLang directives of the WAF.
	Directives []string `json:"directives,omitempty" toml:"directives,omitempty" yaml:"directives,omitempty" export:"true"`
	// CRSEnabled makes the embedded OWASP Core Rule Set files available to the Include directives:
	// @coraza.conf-recommended, @crs-setup.conf.example and @owasp_crs/*.conf.
	CRSEnabled bool `json:"crsEnabled,omitempty" toml:"crsEnabled,omitempty" yaml:"crsEnabled,omitempty" export:"true"`
	// FailOpen lets the requests through when the WAF fails to process them, instead of answering 500.
	FailOpen bool `json:"failOpen,omitempty" toml:"failOpen,omitempty" yaml:"failOpen,omitempty" export:"true"`
}
