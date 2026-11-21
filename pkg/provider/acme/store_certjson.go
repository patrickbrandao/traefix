package acme

import (
	"crypto/ecdsa"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// certJSON is the cert.json document of the traefik-certs project,
// whose consumers (cert-get, cert-watcher) read these exact field names.
// The MD5 sums are checksums the format requires, not a security measure.
type certJSON struct {
	FQDN              string   `json:"fqdn"`
	SANs              []string `json:"sans"`
	IsWildcard        bool     `json:"is_wildcard"`
	SubjectCN         string   `json:"subject_cn"`
	IssuerCN          string   `json:"issuer_cn"`
	IssuerOrg         string   `json:"issuer_org"`
	Serial            string   `json:"serial"`
	NotBefore         string   `json:"not_before"`
	NotBeforeUnix     int64    `json:"not_before_unix"`
	NotAfter          string   `json:"not_after"`
	NotAfterUnix      int64    `json:"not_after_unix"`
	SecondsToExpiry   int64    `json:"seconds_to_expiry"`
	IsCurrentlyValid  bool     `json:"is_currently_valid"`
	FingerprintSHA256 string   `json:"fingerprint_sha256"`
	CertMD5           string   `json:"cert_md5"`
	KeyAlgorithm      string   `json:"key_algorithm"`
	KeyBits           int      `json:"key_bits"`
	SignatureAlgo     string   `json:"signature_algorithm"`
	Source            string   `json:"source"`
	AcmeResolver      string   `json:"acme_resolver"`
	UpdatedAt         string   `json:"updated_at"`
	PEM               certPEM  `json:"pem"`
	Hash              certHash `json:"hash"`

	// Traefix is ignored by traefik-certs, and lets Traefix rebuild its own record from the file.
	Traefix *certJSONTraefix `json:"traefix,omitempty"`
}

type certPEM struct {
	Cert      string `json:"cert"`
	Chain     string `json:"chain"`
	Fullchain string `json:"fullchain"`
	Privkey   string `json:"privkey"`
}

type certHash struct {
	Cert      string `json:"cert"`
	Chain     string `json:"chain"`
	Fullchain string `json:"fullchain"`
	Privkey   string `json:"privkey"`
}

// certJSONTraefix holds what an x509 certificate does not tell:
// the resolver, the domain as the ACME provider tracks it, and the TLS store.
type certJSONTraefix struct {
	Resolver string   `json:"resolver"`
	Main     string   `json:"main"`
	SANs     []string `json:"sans,omitempty"`
	Store    string   `json:"store"`
}

const certSourceACME = "acme"

// newCertJSON builds the cert.json of a certificate for the directory named fqdn.
func newCertJSON(fqdn, resolver string, cert *CertAndStore, now time.Time) (*certJSON, error) {
	fullchain := string(cert.Certificate.Certificate)
	privkey := string(cert.Key)

	leafPEM, chainPEM, leaf, err := decomposePEM(fullchain)
	if err != nil {
		return nil, err
	}

	// The SANs come from the certificate itself, never from the configuration that asked for it.
	sans := leaf.DNSNames
	if len(sans) == 0 && leaf.Subject.CommonName != "" {
		sans = []string{leaf.Subject.CommonName}
	}

	now = now.UTC()
	notBefore := leaf.NotBefore.UTC()
	notAfter := leaf.NotAfter.UTC()

	var keyAlgorithm string
	var keyBits int
	switch pub := leaf.PublicKey.(type) {
	case *rsa.PublicKey:
		keyAlgorithm = "RSA"
		keyBits = pub.N.BitLen()
	case *ecdsa.PublicKey:
		keyAlgorithm = "ECDSA"
		keyBits = pub.Curve.Params().BitSize
	default:
		keyAlgorithm = fmt.Sprintf("%T", leaf.PublicKey)
	}

	var issuerOrg string
	if len(leaf.Issuer.Organization) > 0 {
		issuerOrg = leaf.Issuer.Organization[0]
	}

	issuerCN := leaf.Issuer.CommonName
	if issuerCN == "" {
		issuerCN = issuerOrg
	}

	fingerprint := sha256.Sum256(leaf.Raw)
	certMD5 := md5Hex(leafPEM)

	return &certJSON{
		FQDN:              fqdn,
		SANs:              sans,
		IsWildcard:        slices.ContainsFunc(sans, isWildcard),
		SubjectCN:         leaf.Subject.CommonName,
		IssuerCN:          issuerCN,
		IssuerOrg:         issuerOrg,
		Serial:            fmt.Sprintf("%x", leaf.SerialNumber),
		NotBefore:         notBefore.Format(time.RFC3339),
		NotBeforeUnix:     notBefore.Unix(),
		NotAfter:          notAfter.Format(time.RFC3339),
		NotAfterUnix:      notAfter.Unix(),
		SecondsToExpiry:   max(int64(notAfter.Sub(now).Seconds()), 0),
		IsCurrentlyValid:  !now.Before(notBefore) && !now.After(notAfter),
		FingerprintSHA256: hex.EncodeToString(fingerprint[:]),
		CertMD5:           certMD5,
		KeyAlgorithm:      keyAlgorithm,
		KeyBits:           keyBits,
		SignatureAlgo:     leaf.SignatureAlgorithm.String(),
		Source:            certSourceACME,
		AcmeResolver:      resolver,
		UpdatedAt:         now.Format(time.RFC3339),
		PEM: certPEM{
			Cert:      leafPEM,
			Chain:     chainPEM,
			Fullchain: fullchain,
			Privkey:   privkey,
		},
		Hash: certHash{
			Cert:      certMD5,
			Chain:     md5Hex(chainPEM),
			Fullchain: md5Hex(fullchain),
			Privkey:   md5Hex(privkey),
		},
		Traefix: &certJSONTraefix{
			Resolver: resolver,
			Main:     cert.Domain.Main,
			SANs:     cert.Domain.SANs,
			Store:    cert.Store,
		},
	}, nil
}

// signature identifies what a cert.json says, apart from the fields that depend on the time it was written.
func (c *certJSON) signature() string {
	sig := strings.Join([]string{c.FQDN, c.AcmeResolver, c.Hash.Fullchain, c.Hash.Privkey}, "|")
	if c.Traefix != nil {
		sig += "|" + c.Traefix.Resolver + "|" + c.Traefix.Store + "|" + c.Traefix.Main + "|" + strings.Join(c.Traefix.SANs, ",")
	}

	return sig
}

// decomposePEM splits a certificate bundle into its leaf and the rest of the chain, and parses the leaf.
func decomposePEM(fullchain string) (string, string, *x509.Certificate, error) {
	var blocks []string
	var leaf *x509.Certificate

	rest := []byte(fullchain)
	for len(rest) > 0 {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			continue
		}

		if leaf == nil {
			var err error
			leaf, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return "", "", nil, fmt.Errorf("parsing leaf certificate: %w", err)
			}
		}

		blocks = append(blocks, string(pem.EncodeToMemory(block)))
	}

	if leaf == nil {
		return "", "", nil, errors.New("no certificate in the PEM bundle")
	}

	return blocks[0], strings.Join(blocks[1:], ""), leaf, nil
}

func isWildcard(domain string) bool {
	return strings.HasPrefix(domain, "*.")
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
