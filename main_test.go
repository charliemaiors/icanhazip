package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetIPAddressIPv4(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.1:12345"
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{},
		},
	}

	getIPAddress(w, req)

	resp := w.Result()
	body := w.Body.String()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if strings.TrimSpace(body) != "203.0.113.1" {
		t.Errorf("Expected IP 203.0.113.1, got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressIPv6(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[2001:db8::1]:12345"
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "2001:db8::1" {
		t.Errorf("Expected IP 2001:db8::1, got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressXForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.2, 192.168.1.1")
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{"X-Forwarded-For"},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "203.0.113.5" {
		t.Errorf("Expected IP 203.0.113.5, got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressXRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "198.51.100.10")
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{"X-Real-IP"},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "198.51.100.10" {
		t.Errorf("Expected IP 198.51.100.10, got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressPrivateIncluded(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: true,
			HTTPHeaders:    []string{},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100 (private included), got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressPrivateExcluded(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "192.168.1.100" {
		t.Errorf("Expected fallback to remote addr, got %s", strings.TrimSpace(body))
	}
}

func TestIsPrivateSubnet(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.254", true},
		{"100.64.0.1", true},
		{"100.127.255.254", true},
		{"172.16.0.1", true},
		{"172.31.255.254", true},
		{"192.168.0.1", true},
		{"192.168.255.254", true},
		{"203.0.113.1", false},
		{"198.51.100.1", false},
		{"8.8.8.8", false},
	}

	for _, test := range tests {
		ip := mustParseIP(test.ip)
		result := isPrivateSubnet(ip)
		if result != test.expected {
			t.Errorf("isPrivateSubnet(%s): expected %v, got %v", test.ip, test.expected, result)
		}
	}
}

func TestInRange(t *testing.T) {
	r := ipRange{
		start: mustParseIP("192.168.0.0"),
		end:   mustParseIP("192.168.255.255"),
	}

	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.168.0.0", true},
		{"192.168.1.1", true},
		{"192.168.255.254", true},
		{"192.167.255.255", false},
		{"192.169.0.0", false},
	}

	for _, test := range tests {
		ip := mustParseIP(test.ip)
		result := inRange(r, ip)
		if result != test.expected {
			t.Errorf("inRange(%s): expected %v, got %v", test.ip, test.expected, result)
		}
	}
}

func TestMultipleHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "10.0.0.2")
	req.Header.Set("X-Real-IP", "203.0.113.50")
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{"X-Forwarded-For", "X-Real-IP"},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "203.0.113.50" {
		t.Errorf("Expected IP 203.0.113.50 from X-Real-IP, got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressXForwardedForWithSpaces(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", " 203.0.113.5 , 10.0.0.2 , 192.168.1.1 ")
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{"X-Forwarded-For"},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "203.0.113.5" {
		t.Errorf("Expected IP 203.0.113.5 (trimmed), got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressXForwardedForIPv6(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "2001:db8::1")
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{"X-Forwarded-For"},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "10.0.0.1" {
		t.Errorf("Expected fallback to RemoteAddr IPv4 10.0.0.1 (IPv6-only header skipped), got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressIPv6OnlyRemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[2001:db8::1]:12345"
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "2001:db8::1" {
		t.Errorf("Expected IPv6 2001:db8::1 from RemoteAddr, got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressEmptyHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.1:12345"
	req.Header.Set("X-Forwarded-For", "")
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{"X-Forwarded-For"},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "203.0.113.1" {
		t.Errorf("Expected fallback to RemoteAddr 203.0.113.1, got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressInvalidHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.1:12345"
	req.Header.Set("X-Forwarded-For", "not-an-ip")
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{"X-Forwarded-For"},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "203.0.113.1" {
		t.Errorf("Expected fallback to RemoteAddr 203.0.113.1, got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressHeaderWithPort(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.5:8080, 10.0.0.2")
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{"X-Forwarded-For"},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "203.0.113.5" {
		t.Errorf("Expected IP 203.0.113.5 (port stripped), got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressAllPrivateHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.2, 172.16.0.1")
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{"X-Forwarded-For"},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "10.0.0.1" {
		t.Errorf("Expected fallback to RemoteAddr 10.0.0.1, got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressCGNAT(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "100.64.0.1:12345"
	w := httptest.NewRecorder()

	config = Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: false,
			HTTPHeaders:    []string{},
		},
	}

	getIPAddress(w, req)

	body := w.Body.String()
	if strings.TrimSpace(body) != "100.64.0.1" {
		t.Errorf("Expected CGNAT IP 100.64.0.1 (treated as private when IncludePrivate=false), got %s", strings.TrimSpace(body))
	}
}

func TestGetIPAddressDocumentationRanges(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{"TEST-NET-1", "192.0.2.1", "192.0.2.1"},
		{"TEST-NET-2", "198.18.0.1", "198.18.0.1"},
		{"TEST-NET-3", "203.0.113.1", "203.0.113.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.ip + ":12345"
			w := httptest.NewRecorder()

			config = Config{
				Server: Server{
					Port: 8091,
					Host: "",
				},
				Results: Result{
					IncludePrivate: false,
					HTTPHeaders:    []string{},
				},
			}

			getIPAddress(w, req)

			body := w.Body.String()
			if strings.TrimSpace(body) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, strings.TrimSpace(body))
			}
		})
	}
}

func mustParseIP(ip string) net.IP {
	result := net.ParseIP(ip)
	if result == nil {
		panic("invalid IP: " + ip)
	}
	return result
}
