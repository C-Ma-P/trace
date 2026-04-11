package phoneintake

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	caCertFile     = "ca-cert.pem"
	caKeyFile      = "ca-key.pem"
	serverCertFile = "server-cert.pem"
	serverKeyFile  = "server-key.pem"

	caCommonName = "Trace Local CA"

	caLifetime     = 10 * 365 * 24 * time.Hour
	serverLifetime = 2 * 365 * 24 * time.Hour
	renewBefore    = 30 * 24 * time.Hour
)

// PKI holds the persistent certificate material for the phone intake server.
type PKI struct {
	CACertPEM []byte
	TLSConfig *tls.Config
}

func LoadOrCreatePKI(dir string, lanIP string) (*PKI, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("phone-intake pki: create dir: %w", err)
	}

	caCert, caKey, caCertPEM, err := loadOrCreateCA(dir)
	if err != nil {
		return nil, err
	}

	serverTLSCert, err := loadOrCreateServerCert(dir, caCert, caKey, lanIP)
	if err != nil {
		return nil, err
	}

	return &PKI{
		CACertPEM: caCertPEM,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{serverTLSCert},
			MinVersion:   tls.VersionTLS12,
		},
	}, nil
}

func loadOrCreateCA(dir string) (*x509.Certificate, *ecdsa.PrivateKey, []byte, error) {
	certPath := filepath.Join(dir, caCertFile)
	keyPath := filepath.Join(dir, caKeyFile)

	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)

	if certErr == nil && keyErr == nil {
		cert, key, err := decodeCertKey(certPEM, keyPEM)
		if err == nil && time.Now().Before(cert.NotAfter.Add(-renewBefore)) {
			return cert, key, certPEM, nil
		}
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("phone-intake pki: generate CA key: %w", err)
	}
	serial, err := newSerial()
	if err != nil {
		return nil, nil, nil, err
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: caCommonName, Organization: []string{"Trace"}},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(caLifetime),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("phone-intake pki: sign CA cert: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("phone-intake pki: marshal CA key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return nil, nil, nil, fmt.Errorf("phone-intake pki: write CA cert: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, nil, nil, fmt.Errorf("phone-intake pki: write CA key: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("phone-intake pki: parse CA cert: %w", err)
	}
	return cert, key, certPEM, nil
}

func loadOrCreateServerCert(dir string, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, lanIP string) (tls.Certificate, error) {
	certPath := filepath.Join(dir, serverCertFile)
	keyPath := filepath.Join(dir, serverKeyFile)

	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		cert, _, err := decodeCertKey(certPEM, keyPEM)
		issuedByCA := err == nil && bytes.Equal(cert.RawIssuer, caCert.RawSubject)
		notExpiringSoon := err == nil && time.Now().Before(cert.NotAfter.Add(-renewBefore))
		if issuedByCA && notExpiringSoon {
			tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
			if err == nil {
				return tlsCert, nil
			}
		}
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("phone-intake pki: generate server key: %w", err)
	}
	serial, err := newSerial()
	if err != nil {
		return tls.Certificate{}, err
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: stableHostname},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(serverLifetime),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{stableHostname, "localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	if ip := net.ParseIP(lanIP); ip != nil {
		tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("phone-intake pki: sign server cert: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("phone-intake pki: marshal server key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return tls.Certificate{}, fmt.Errorf("phone-intake pki: write server cert: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return tls.Certificate{}, fmt.Errorf("phone-intake pki: write server key: %w", err)
	}

	return tls.X509KeyPair(certPEM, keyPEM)
}

func decodeCertKey(certPEM, keyPEM []byte) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, errors.New("no cert PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, errors.New("no key PEM block")
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func newSerial() (*big.Int, error) {
	n, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("phone-intake pki: generate serial: %w", err)
	}
	return n, nil
}
