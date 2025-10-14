package proxy

import (
	"os"
	"testing"
)

func TestConfigFromEnvironment(t *testing.T) {
	// Save original environment variables and restore them after the test
	originalEnv := make(map[string]string)
	keysToSave := []string{
		"TLS_NAME", "CA_NAME", "CERT_CA_NAMESPACE", "CERT_CA_NAME",
		"SECRET", "PROXY_PORT", "PEER_PORT", "HTTPS_PORT", "DEBUG",
	}
	for _, key := range keysToSave {
		originalEnv[key] = os.Getenv(key)
		_ = os.Unsetenv(key) // Ensure a clean slate for each test case
	}
	defer func() {
		for _, key := range keysToSave {
			_ = os.Setenv(key, originalEnv[key])
		}
	}()

	tests := []struct {
		name        string
		setupEnv    func(t *testing.T)
		expectError bool
		expected    *Config
	}{
		{
			name: "Success with all variables set",
			setupEnv: func(t *testing.T) {
				t.Setenv("TLS_NAME", "test-tls")
				t.Setenv("CA_NAME", "test-ca")
				t.Setenv("CERT_CA_NAMESPACE", "test-namespace")
				t.Setenv("CERT_CA_NAME", "test-cert-ca")
				t.Setenv("SECRET", "test-secret")
				t.Setenv("PROXY_PORT", "8080")
				t.Setenv("PEER_PORT", "8081")
				t.Setenv("HTTPS_PORT", "8443")
				t.Setenv("DEBUG", "true")
			},
			expectError: false,
			expected: &Config{
				TLSName:         "test-tls",
				CAName:          "test-ca",
				CertCANamespace: "test-namespace",
				CertCAName:      "test-cert-ca",
				Secret:          "test-secret",
				ProxyPort:       8080,
				PeerPort:        8081,
				HTTPSPort:       8443,
				Debug:           true,
			},
		},
		{
			name: "Success without DEBUG",
			setupEnv: func(t *testing.T) {
				t.Setenv("TLS_NAME", "test-tls")
				t.Setenv("CA_NAME", "test-ca")
				t.Setenv("CERT_CA_NAMESPACE", "test-namespace")
				t.Setenv("CERT_CA_NAME", "test-cert-ca")
				t.Setenv("SECRET", "test-secret")
				t.Setenv("PROXY_PORT", "8080")
				t.Setenv("PEER_PORT", "8081")
				t.Setenv("HTTPS_PORT", "8443")
				_ = os.Unsetenv("DEBUG")
			},
			expectError: false,
			expected: &Config{
				TLSName:         "test-tls",
				CAName:          "test-ca",
				CertCANamespace: "test-namespace",
				CertCAName:      "test-cert-ca",
				Secret:          "test-secret",
				ProxyPort:       8080,
				PeerPort:        8081,
				HTTPSPort:       8443,
				Debug:           false,
			},
		},
		{
			name: "Missing TLS_NAME",
			setupEnv: func(t *testing.T) {
				t.Setenv("CA_NAME", "test-ca")
				t.Setenv("CERT_CA_NAMESPACE", "test-namespace")
				t.Setenv("CERT_CA_NAME", "test-cert-ca")
				t.Setenv("SECRET", "test-secret")
				t.Setenv("PROXY_PORT", "8080")
				t.Setenv("PEER_PORT", "8081")
				t.Setenv("HTTPS_PORT", "8443")
			},
			expectError: true,
		},
		{
			name: "Invalid PROXY_PORT",
			setupEnv: func(t *testing.T) {
				t.Setenv("TLS_NAME", "test-tls")
				t.Setenv("CA_NAME", "test-ca")
				t.Setenv("CERT_CA_NAMESPACE", "test-namespace")
				t.Setenv("CERT_CA_NAME", "test-cert-ca")
				t.Setenv("SECRET", "test-secret")
				t.Setenv("PROXY_PORT", "not-a-port") // Invalid port
				t.Setenv("PEER_PORT", "8081")
				t.Setenv("HTTPS_PORT", "8443")
			},
			expectError: true,
		},
		{
			name: "PROXY_PORT zero",
			setupEnv: func(t *testing.T) {
				t.Setenv("TLS_NAME", "test-tls")
				t.Setenv("CA_NAME", "test-ca")
				t.Setenv("CERT_CA_NAMESPACE", "test-namespace")
				t.Setenv("CERT_CA_NAME", "test-cert-ca")
				t.Setenv("SECRET", "test-secret")
				t.Setenv("PROXY_PORT", "0") // Invalid port
				t.Setenv("PEER_PORT", "8081")
				t.Setenv("HTTPS_PORT", "8443")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset environment for each test run
			for _, key := range keysToSave {
				_ = os.Unsetenv(key)
			}

			tt.setupEnv(t)

			config, err := ConfigFromEnvironment()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error but got: %v", err)
				}
				if config == nil {
					t.Fatalf("Expected config but got nil")
				}
				if config.TLSName != tt.expected.TLSName {
					t.Errorf("TLSName mismatch: got %s, want %s", config.TLSName, tt.expected.TLSName)
				}
				if config.CAName != tt.expected.CAName {
					t.Errorf("CAName mismatch: got %s, want %s", config.CAName, tt.expected.CAName)
				}
				if config.CertCANamespace != tt.expected.CertCANamespace {
					t.Errorf("CertCANamespace mismatch: got %s, want %s", config.CertCANamespace, tt.expected.CertCANamespace)
				}
				if config.CertCAName != tt.expected.CertCAName {
					t.Errorf("CertCAName mismatch: got %s, want %s", config.CertCAName, tt.expected.CertCAName)
				}
				if config.Secret != tt.expected.Secret {
					t.Errorf("Secret mismatch: got %s, want %s", config.Secret, tt.expected.Secret)
				}
				if config.ProxyPort != tt.expected.ProxyPort {
					t.Errorf("ProxyPort mismatch: got %d, want %d", config.ProxyPort, tt.expected.ProxyPort)
				}
				if config.PeerPort != tt.expected.PeerPort {
					t.Errorf("PeerPort mismatch: got %d, want %d", config.PeerPort, tt.expected.PeerPort)
				}
				if config.HTTPSPort != tt.expected.HTTPSPort {
					t.Errorf("HTTPSPort mismatch: got %d, want %d", config.HTTPSPort, tt.expected.HTTPSPort)
				}
				if config.Debug != tt.expected.Debug {
					t.Errorf("Debug mismatch: got %t, want %t", config.Debug, tt.expected.Debug)
				}
			}
		})
	}
}
