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
