package static

import (
	"time"

	ptypes "github.com/traefik/paerser/types"
)

// DefaultFingerprintHeaderPrefix is the default prefix of the headers carrying the connection data and fingerprints.
const DefaultFingerprintHeaderPrefix = "X-Traefix-"

// Fingerprint configures the identification and fingerprinting of the connections and requests of an entry point.
type Fingerprint struct {
	HeaderPrefix string             `description:"Prefix of the request headers carrying the connection data and fingerprints. The client request headers with this prefix are removed." json:"headerPrefix,omitempty" toml:"headerPrefix,omitempty" yaml:"headerPrefix,omitempty" export:"true"`
	Export       *FingerprintExport `description:"Exports the connection and request records to an HTTP endpoint." json:"export,omitempty" toml:"export,omitempty" yaml:"export,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (f *Fingerprint) SetDefaults() {
	f.HeaderPrefix = DefaultFingerprintHeaderPrefix
}

// FingerprintExport configures the export of the connection and request records.
type FingerprintExport struct {
	URL                  string            `description:"URL receiving the batches of records, sent with POST requests." json:"url,omitempty" toml:"url,omitempty" yaml:"url,omitempty"`
	Headers              map[string]string `description:"Headers sent with each batch. Their values are treated as secrets." json:"headers,omitempty" toml:"headers,omitempty" yaml:"headers,omitempty" loggable:"false"`
	RawBytes             bool              `description:"Exports the raw bytes of the TLS ClientHello and of the HTTP/2 connection prefix." json:"rawBytes,omitempty" toml:"rawBytes,omitempty" yaml:"rawBytes,omitempty" export:"true"`
	Fields               []string          `description:"Request headers whose values are exported. A trailing * matches the names starting with the given prefix." json:"fields,omitempty" toml:"fields,omitempty" yaml:"fields,omitempty" export:"true"`
	AllowSensitiveFields bool              `description:"Allows exporting the values of the Cookie, Authorization, Proxy-Authorization and Set-Cookie headers listed in fields." json:"allowSensitiveFields,omitempty" toml:"allowSensitiveFields,omitempty" yaml:"allowSensitiveFields,omitempty" export:"true"`
	QueueSize            int               `description:"Maximum number of records waiting to be sent." json:"queueSize,omitempty" toml:"queueSize,omitempty" yaml:"queueSize,omitempty" export:"true"`
	QueueMaxBytes        int               `description:"Maximum size in bytes of the records waiting to be sent." json:"queueMaxBytes,omitempty" toml:"queueMaxBytes,omitempty" yaml:"queueMaxBytes,omitempty" export:"true"`
	BatchSize            int               `description:"Number of records that triggers sending a batch, up to 1024." json:"batchSize,omitempty" toml:"batchSize,omitempty" yaml:"batchSize,omitempty" export:"true"`
	FlushInterval        ptypes.Duration   `description:"Maximum time a record waits before its batch is sent." json:"flushInterval,omitempty" toml:"flushInterval,omitempty" yaml:"flushInterval,omitempty" export:"true"`
	SendTimeout          ptypes.Duration   `description:"Timeout of the request sending a batch." json:"sendTimeout,omitempty" toml:"sendTimeout,omitempty" yaml:"sendTimeout,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (e *FingerprintExport) SetDefaults() {
	e.Fields = []string{"User-Agent", "Accept", "Accept-Language", "Accept-Encoding", "Sec-CH-UA*", "Sec-Fetch-*"}
	e.QueueSize = 4096
	e.QueueMaxBytes = 16 << 20
	e.BatchSize = 64
	e.FlushInterval = ptypes.Duration(100 * time.Millisecond)
	e.SendTimeout = ptypes.Duration(2 * time.Second)
}
