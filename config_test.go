package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.Server.Port != 8091 {
		t.Errorf("Expected default port 8091, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "" {
		t.Errorf("Expected default host empty, got %s", cfg.Server.Host)
	}
	if cfg.Results.IncludePrivate != true {
		t.Errorf("Expected IncludePrivate true, got %v", cfg.Results.IncludePrivate)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 9090
  host: 127.0.0.1
  enable_proxy_protocol: true
results:
  include_private: false
  http_headers:
    - "X-Forwarded-For"
    - "X-Real-IP"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	cfg := Config{}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", cfg.Server.Host)
	}
	if cfg.Server.EnableProxyProtocol != true {
		t.Errorf("Expected EnableProxyProtocol true, got %v", cfg.Server.EnableProxyProtocol)
	}
	if cfg.Results.IncludePrivate != false {
		t.Errorf("Expected IncludePrivate false, got %v", cfg.Results.IncludePrivate)
	}
	if len(cfg.Results.HTTPHeaders) != 2 {
		t.Errorf("Expected 2 HTTP headers, got %d", len(cfg.Results.HTTPHeaders))
	}
}

func TestLoadConfigWithTLS(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 443
  tls:
    cert_file: /etc/icanhazip/cert.pem
    key_file: /etc/icanhazip/key.pem
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	cfg := Config{}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if cfg.Server.TLS == nil {
		t.Fatal("Expected TLS config to be set")
	}
	if cfg.Server.TLS.CertFile != "/etc/icanhazip/cert.pem" {
		t.Errorf("Expected cert_file /etc/icanhazip/cert.pem, got %s", cfg.Server.TLS.CertFile)
	}
	if cfg.Server.TLS.KeyFile != "/etc/icanhazip/key.pem" {
		t.Errorf("Expected key_file /etc/icanhazip/key.pem, got %s", cfg.Server.TLS.KeyFile)
	}
}

func TestLoadConfigWithACME(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 443
  tls:
    acme:
      email: test@example.com
      domains:
        - example.com
        - www.example.com
      acme_directory_url: https://acme-v02.api.letsencrypt.org/directory
      http01_port: "80"
      tlsalpn01_port: "443"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	cfg := Config{}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if cfg.Server.TLS == nil {
		t.Fatal("Expected TLS config to be set")
	}
	if cfg.Server.TLS.Acme == nil {
		t.Fatal("Expected ACME config to be set")
	}
	if cfg.Server.TLS.Acme.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", cfg.Server.TLS.Acme.Email)
	}
	if len(cfg.Server.TLS.Acme.Domains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(cfg.Server.TLS.Acme.Domains))
	}
	if cfg.Server.TLS.Acme.HTTP01Port != "80" {
		t.Errorf("Expected HTTP01 port 80, got %s", cfg.Server.TLS.Acme.HTTP01Port)
	}
}

func TestLoadConfigWithHTTPHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
results:
  include_private: true
  http_headers:
    - "X-Forwarded-For"
    - "X-Real-IP"
    - "CF-Connecting-IP"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	cfg := Config{}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if len(cfg.Results.HTTPHeaders) != 3 {
		t.Errorf("Expected 3 HTTP headers, got %d", len(cfg.Results.HTTPHeaders))
	}
	expectedHeaders := []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"}
	for i, h := range expectedHeaders {
		if cfg.Results.HTTPHeaders[i] != h {
			t.Errorf("Expected header %s at position %d, got %s", h, i, cfg.Results.HTTPHeaders[i])
		}
	}
}
