package selfcert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCertCoversRequiresLANIP(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"test"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("172.19.0.2")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}}

	if !certCovers(cert, []net.IP{net.ParseIP("127.0.0.1")}, []string{"localhost"}) {
		t.Fatal("expected cover for localhost IP")
	}
	if certCovers(cert, []net.IP{net.ParseIP("192.0.2.10")}, []string{"localhost"}) {
		t.Fatal("docker-only cert must not cover extra LAN IP")
	}
}

func TestEnsureCertAddsExtraIPsAndRegenerates(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TLS_EXTRA_IPS", "")
	t.Setenv("TLS_EXTRA_HOSTS", "")

	c1, err := EnsureCert(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !certCovers(c1, []net.IP{net.ParseIP("127.0.0.1")}, []string{"localhost"}) {
		t.Fatal("fresh cert should cover localhost")
	}

	t.Setenv("TLS_EXTRA_IPS", "192.0.2.10")
	c2, err := EnsureCert(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !certCovers(c2, []net.IP{net.ParseIP("192.0.2.10")}, []string{"localhost"}) {
		t.Fatal("regenerated cert should include TLS_EXTRA_IPS")
	}
	// Files on disk updated
	if _, err := os.Stat(filepath.Join(dir, "cert.pem")); err != nil {
		t.Fatal(err)
	}
}

func TestDesiredIPsParsesEnv(t *testing.T) {
	t.Setenv("TLS_EXTRA_IPS", "10.0.0.5, 192.168.1.10")
	ips := desiredIPs()
	want := map[string]bool{"10.0.0.5": false, "192.168.1.10": false}
	for _, ip := range ips {
		if _, ok := want[ip.String()]; ok {
			want[ip.String()] = true
		}
	}
	for k, ok := range want {
		if !ok {
			t.Fatalf("missing %s in desiredIPs", k)
		}
	}
}

func pemLeaf(t *testing.T, cert tls.Certificate) *x509.Certificate {
	t.Helper()
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	return leaf
}

func TestEnsureCertWritesPEM(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TLS_EXTRA_IPS", "10.1.2.3")
	cert, err := EnsureCert(dir)
	if err != nil {
		t.Fatal(err)
	}
	leaf := pemLeaf(t, cert)
	found := false
	for _, ip := range leaf.IPAddresses {
		if ip.Equal(net.ParseIP("10.1.2.3")) {
			found = true
		}
	}
	if !found {
		t.Fatalf("SAN missing 10.1.2.3: %v", leaf.IPAddresses)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "cert.pem"))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		t.Fatal("expected PEM block")
	}
}

func TestEnsureCertValidityOneYear(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TLS_EXTRA_IPS", "")
	t.Setenv("TLS_EXTRA_HOSTS", "")
	cert, err := EnsureCert(dir)
	if err != nil {
		t.Fatal(err)
	}
	leaf := pemLeaf(t, cert)
	want := time.Now().Add(certValidity)
	delta := leaf.NotAfter.Sub(want)
	if delta < -2*time.Minute || delta > 2*time.Minute {
		t.Fatalf("NotAfter %v, want about %v (delta %v)", leaf.NotAfter, want, delta)
	}
}
