package main

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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func generateTestCertificate(t *testing.T) (certFile, keyFile string, cleanup func()) {
	t.Helper()

	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatalf("Failed to write cert: %v", err)
	}

	keyOut, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		t.Fatalf("Failed to write key: %v", err)
	}

	return certPath, keyPath, func() {
		os.RemoveAll(tmpDir)
	}
}

func TestTLSServerWithCertAndKey(t *testing.T) {
	certFile, keyFile, _ := generateTestCertificate(t)

	config = Config{
		Server: Server{
			Port: 0,
			Host: "127.0.0.1",
			TLS: &TLS{
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		},
		Results: Result{
			IncludePrivate: false,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", getIPAddress)

	server := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: mux,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			t.Errorf("Failed to load certificate: %v", err)
			return
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		server.TLSConfig = tlsConfig

		if err := server.ServeTLS(listener, "", ""); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	}()

	defer server.Close()

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to load certificate for client: %v", err)
	}

	certPool := x509.NewCertPool()
	parsedCert, _ := x509.ParseCertificate(cert.Certificate[0])
	certPool.AddCert(parsedCert)

	tlsClientConfig := &tls.Config{
		RootCAs: certPool,
	}

	tr := &http.Transport{
		TLSClientConfig: tlsClientConfig,
	}
	client := &http.Client{Transport: tr}

	addr := listener.Addr().String()
	req, err := http.NewRequest("GET", "https://"+addr+"/", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.RemoteAddr = "203.0.113.1:12345"

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestTLSConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		tlsConfig     *TLS
		expectValid   bool
		expectACME    bool
		expectCertKey bool
	}{
		{
			name: "CertAndKey",
			tlsConfig: &TLS{
				CertFile: "/path/to/cert.pem",
				KeyFile:  "/path/to/key.pem",
			},
			expectValid:   true,
			expectACME:    false,
			expectCertKey: true,
		},
		{
			name: "ACMEOnly",
			tlsConfig: &TLS{
				Acme: &Acme{
					Email:   "test@example.com",
					Domains: []string{"example.com"},
				},
			},
			expectValid:   true,
			expectACME:    true,
			expectCertKey: false,
		},
		{
			name:          "NoTLS",
			tlsConfig:     nil,
			expectValid:   true,
			expectACME:    false,
			expectCertKey: false,
		},
		{
			name: "BothCertKeyAndACME",
			tlsConfig: &TLS{
				CertFile: "/path/to/cert.pem",
				KeyFile:  "/path/to/key.pem",
				Acme: &Acme{
					Email: "test@example.com",
				},
			},
			expectValid:   true,
			expectACME:    true,
			expectCertKey: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Server: Server{
					TLS: tt.tlsConfig,
				},
			}

			hasCertKey := cfg.Server.TLS != nil && cfg.Server.TLS.CertFile != "" && cfg.Server.TLS.KeyFile != ""
			hasACME := cfg.Server.TLS != nil && cfg.Server.TLS.Acme != nil

			if hasCertKey != tt.expectCertKey {
				t.Errorf("Expected cert/key=%v, got %v", tt.expectCertKey, hasCertKey)
			}
			if hasACME != tt.expectACME {
				t.Errorf("Expected ACME=%v, got %v", tt.expectACME, hasACME)
			}
		})
	}
}

func TestACMEConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		acmeConfig    *Acme
		expectValid   bool
		expectedEmail string
	}{
		{
			name: "ValidACME",
			acmeConfig: &Acme{
				Email:   "test@example.com",
				Domains: []string{"example.com", "www.example.com"},
			},
			expectValid:   true,
			expectedEmail: "test@example.com",
		},
		{
			name: "ACMENoEmail",
			acmeConfig: &Acme{
				Domains: []string{"example.com"},
			},
			expectValid:   true,
			expectedEmail: "",
		},
		{
			name:          "NilACME",
			acmeConfig:    nil,
			expectValid:   true,
			expectedEmail: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Server: Server{
					TLS: &TLS{
						Acme: tt.acmeConfig,
					},
				},
			}

			if tt.acmeConfig != nil {
				if cfg.Server.TLS.Acme.Email != tt.expectedEmail {
					t.Errorf("Expected email %s, got %s", tt.expectedEmail, cfg.Server.TLS.Acme.Email)
				}
			}
		})
	}
}

func TestHTTPSServerHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.1:12345"
	req.TLS = &tls.ConnectionState{
		Version:     tls.VersionTLS13,
		CipherSuite: tls.TLS_AES_256_GCM_SHA384,
	}
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 443,
			Host: "0.0.0.0",
			TLS: &TLS{
				CertFile: "/path/to/cert.pem",
				KeyFile:  "/path/to/key.pem",
			},
		},
		Results: Result{
			IncludePrivate: false,
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "203.0.113.1" {
		t.Errorf("Expected IP 203.0.113.1 over HTTPS, got %s", strings.TrimSpace(body))
	}
}
