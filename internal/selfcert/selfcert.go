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
	"strings"
	"time"
)

// certValidity is how long newly minted self-signed certs last. Existing
// files are reused until SANs change, even if they were minted with a longer life.
const certValidity = 365 * 24 * time.Hour

// EnsureCert returns a tls.Certificate for HTTPS. It loads from certDir if
// files exist and cover the desired hostnames/IPs; otherwise generates a
// self-signed cert valid for one year (localhost + container/host IPs +
// TLS_EXTRA_IPS / TLS_EXTRA_HOSTS), then saves it for reuse.
//
// TLS_EXTRA_IPS is required for Docker bridge networking: inside the
// container, localIPs() only sees the bridge address, not the LAN IP phones use.
func EnsureCert(certDir string) (tls.Certificate, error) {
	certPath := filepath.Join(certDir, "cert.pem")
	keyPath := filepath.Join(certDir, "key.pem")
	wantIPs := desiredIPs()
	wantDNS := desiredDNSNames()

	if cert, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		if certCovers(cert, wantIPs, wantDNS) {
			return cert, nil
		}
		// Stale SANs (e.g. cert minted with only the Docker bridge IP) — regenerate.
		_ = os.Remove(certPath)
		_ = os.Remove(keyPath)
	}

	if err := os.MkdirAll(certDir, 0700); err != nil {
		return tls.Certificate{}, err
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"Sonix Self-Signed"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(certValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     wantDNS,
		IPAddresses:  wantIPs,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	_ = os.WriteFile(certPath, certPEM, 0600)
	_ = os.WriteFile(keyPath, keyPEM, 0600)

	return tls.X509KeyPair(certPEM, keyPEM)
}

func desiredDNSNames() []string {
	names := []string{"localhost"}
	for _, p := range strings.Split(os.Getenv("TLS_EXTRA_HOSTS"), ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			names = append(names, p)
		}
	}
	return names
}

func desiredIPs() []net.IP {
	ips := localIPs()
	for _, p := range strings.Split(os.Getenv("TLS_EXTRA_IPS"), ",") {
		p = strings.TrimSpace(p)
		if ip := net.ParseIP(p); ip != nil {
			ips = append(ips, ip)
		}
	}
	return uniqueIPs(ips)
}

func uniqueIPs(in []net.IP) []net.IP {
	seen := make(map[string]struct{}, len(in))
	out := make([]net.IP, 0, len(in))
	for _, ip := range in {
		if ip == nil {
			continue
		}
		k := ip.String()
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, ip)
	}
	return out
}

func certCovers(cert tls.Certificate, wantIPs []net.IP, wantDNS []string) bool {
	if len(cert.Certificate) == 0 {
		return false
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return false
	}
	haveIP := make(map[string]struct{}, len(leaf.IPAddresses))
	for _, ip := range leaf.IPAddresses {
		haveIP[ip.String()] = struct{}{}
	}
	for _, ip := range wantIPs {
		if _, ok := haveIP[ip.String()]; !ok {
			return false
		}
	}
	haveDNS := make(map[string]struct{}, len(leaf.DNSNames))
	for _, d := range leaf.DNSNames {
		haveDNS[strings.ToLower(d)] = struct{}{}
	}
	for _, d := range wantDNS {
		if _, ok := haveDNS[strings.ToLower(d)]; !ok {
			return false
		}
	}
	return true
}

// localIPs returns localhost + all non-loopback IPs on the machine.
func localIPs() []net.IP {
	ips := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			ips = append(ips, ipNet.IP)
		}
	}
	return ips
}
