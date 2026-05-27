// Package security generates and handles the self-signed TLS identity used for
// LocalSend's encrypted (HTTPS) mode.
//
// The LocalSend fingerprint is the SHA-256 of the certificate's DER bytes,
// encoded as uppercase hex (verified against the official client's stored
// certificateHash). Peers do not validate the certificate chain; they pin this
// fingerprint, which is advertised in the discovery announce.
package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"strings"
	"time"
)

// Identity is a self-signed certificate, its key, and its LocalSend fingerprint.
type Identity struct {
	CertPEM     string
	KeyPEM      string
	Fingerprint string
}

// Fingerprint returns the LocalSend fingerprint of a DER-encoded certificate:
// uppercase hex of its SHA-256.
func Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// Generate creates a fresh RSA-2048 self-signed certificate matching the shape
// the official LocalSend client uses (CN "LocalSend User", ~10-year validity).
func Generate() (Identity, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return Identity{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return Identity{}, err
	}
	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "LocalSend User"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return Identity{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return Identity{
		CertPEM:     string(certPEM),
		KeyPEM:      string(keyPEM),
		Fingerprint: Fingerprint(der),
	}, nil
}
