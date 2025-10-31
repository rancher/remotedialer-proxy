package proxy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFromEnvironment(t *testing.T) {
	keysToSave := []string{
		"TLS_NAME", "CA_NAME", "CERT_CA_NAMESPACE", "CERT_CA_NAME",
		"SECRET", "PROXY_PORT", "PEER_PORT", "HTTPS_PORT", "DEBUG",
	}

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
				require.Error(t, err, "Expected an error but got none")
			} else {
				require.NoError(t, err, "Did not expect an error but got: %v", err)
				require.NotNil(t, config, "Expected config but got nil")
				assert.Equal(t, tt.expected.TLSName, config.TLSName, "TLSName mismatch")
				assert.Equal(t, tt.expected.CAName, config.CAName, "CAName mismatch")
				assert.Equal(t, tt.expected.CertCANamespace, config.CertCANamespace, "CertCANamespace mismatch")
				assert.Equal(t, tt.expected.CertCAName, config.CertCAName, "CertCAName mismatch")
				assert.Equal(t, tt.expected.Secret, config.Secret, "Secret mismatch")
				assert.Equal(t, tt.expected.ProxyPort, config.ProxyPort, "ProxyPort mismatch")
				assert.Equal(t, tt.expected.PeerPort, config.PeerPort, "PeerPort mismatch")
				assert.Equal(t, tt.expected.HTTPSPort, config.HTTPSPort, "HTTPSPort mismatch")
				assert.Equal(t, tt.expected.Debug, config.Debug, "Debug mismatch")
			}
		})
	}
}
