package koanfuri

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewKoanfURI(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()

	jsonFile := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(jsonFile, []byte(`{"key": "value"}`), 0644)
	require.NoError(t, err)

	yamlFile := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(yamlFile, []byte("key: value"), 0644)
	require.NoError(t, err)

	tomlFile := filepath.Join(tmpDir, "config.toml")
	err = os.WriteFile(tomlFile, []byte(`key = "value"`), 0644)
	require.NoError(t, err)

	// Start test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"key": "value"}`))
	}))
	defer ts.Close()

	tests := []struct {
		name           string
		uri            string
		expectedFormat string
		expectError    bool
		skipNoEnvVars  bool // Skip test if required env vars are not set
	}{
		{
			name:           "local JSON file",
			uri:            "file://" + jsonFile,
			expectedFormat: "json",
			expectError:    false,
		},
		{
			name:           "local YAML file",
			uri:            "file://" + yamlFile,
			expectedFormat: "yaml",
			expectError:    false,
		},
		{
			name:           "local TOML file",
			uri:            "file://" + tomlFile,
			expectedFormat: "toml",
			expectError:    false,
		},
		{
			name:           "HTTP JSON",
			uri:            ts.URL,
			expectedFormat: "json",
			expectError:    false,
		},
		{
			name:           "HTTP with JSON hint",
			uri:            "http+json://" + ts.URL[7:], // Remove "http://" prefix
			expectedFormat: "json",
			expectError:    false,
		},
		{
			name:           "HTTPS with YAML hint",
			uri:            "https+yaml://example.com/config",
			expectedFormat: "yaml",
			expectError:    true, // Will fail to connect but format is detected
		},
		{
			name:           "S3 with JSON hint",
			uri:            "s3+json://mybucket/path/to/config.json",
			expectedFormat: "json",
			expectError:    true, // Will fail without AWS credentials
			skipNoEnvVars:  true,
		},
		{
			name:           "AppConfig with YAML hint",
			uri:            "appconfig+yaml://myapp/prod/web",
			expectedFormat: "yaml",
			expectError:    true, // Will fail without AWS credentials
			skipNoEnvVars:  true,
		},
		{
			name:           "Vault with JSON hint",
			uri:            "vault+json://secret/data/myapp",
			expectedFormat: "json",
			expectError:    true, // Will fail without Vault credentials
			skipNoEnvVars:  true,
		},
		{
			name:           "Consul with JSON hint",
			uri:            "consul+json://myapp/config",
			expectedFormat: "json",
			expectError:    true, // Will fail without Consul credentials
			skipNoEnvVars:  true,
		},
		{
			name:        "invalid URI",
			uri:         "invalid://test",
			expectError: true,
		},
		{
			name:        "nonexistent file",
			uri:         "file:///nonexistent.json",
			expectError: true,
		},
		{
			name:        "invalid S3 URI format",
			uri:         "s3://",
			expectError: true,
		},
		{
			name:        "invalid AppConfig URI format",
			uri:         "appconfig://myapp",
			expectError: true,
		},
		{
			name:        "invalid Vault URI format",
			uri:         "vault://",
			expectError: true,
		},
		{
			name:        "invalid Consul URI format",
			uri:         "consul://",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that require env vars if they're not set
			if tt.skipNoEnvVars {
				switch {
				case strings.HasPrefix(tt.uri, "s3://"), strings.HasPrefix(tt.uri, "appconfig://"):
					if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
						t.Skip("Skipping test: AWS credentials not set")
					}
				case strings.HasPrefix(tt.uri, "vault://"):
					if os.Getenv("VAULT_TOKEN") == "" {
						t.Skip("Skipping test: Vault token not set")
					}
				case strings.HasPrefix(tt.uri, "consul://"):
					if os.Getenv("CONSUL_HTTP_TOKEN") == "" {
						t.Skip("Skipping test: Consul token not set")
					}
				}
			}

			k, err := NewKoanfURI(tt.uri)
			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, k)
			require.Equal(t, tt.expectedFormat, k.GetDataFormat())

			// Verify the loaded configuration
			value := k.GetKonfig().String("key")
			require.Equal(t, "value", value)
		})
	}
}

func TestStdinHandling(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tests := []struct {
		name        string
		input       string
		uri         string
		expectError bool
	}{
		{
			name:        "JSON from stdin using dash",
			input:       `{"key": "value"}`,
			uri:         "-",
			expectError: false,
		},
		{
			name:        "YAML from stdin using stdin",
			input:       "key: value",
			uri:         "stdin",
			expectError: false,
		},
		{
			name:        "invalid JSON from stdin",
			input:       "invalid json",
			uri:         "-",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a pipe and write test input
			r, w, err := os.Pipe()
			require.NoError(t, err)
			os.Stdin = r

			go func() {
				w.Write([]byte(tt.input))
				w.Close()
			}()

			k, err := NewKoanfURI(tt.uri)
			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, k)

			// Verify the loaded configuration
			value := k.GetKonfig().String("key")
			require.Equal(t, "value", value)
		})
	}
}

func TestFormatDetection(t *testing.T) {
	tests := []struct {
		name           string
		content        []byte
		expectedFormat string
	}{
		{
			name:           "JSON detection",
			content:        []byte(`{"key": "value"}`),
			expectedFormat: "json",
		},
		{
			name:           "YAML detection",
			content:        []byte("key: value"),
			expectedFormat: "yaml",
		},
		{
			name:           "TOML detection",
			content:        []byte("[section]\nkey = \"value\""),
			expectedFormat: "toml",
		},
		{
			name:           "HCL detection",
			content:        []byte(`resource "aws_instance" "example" {}`),
			expectedFormat: "hcl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &KoanfURI{
				uri: &url.URL{Path: "test.conf"},
			}
			format := k.detectFormat(tt.content)
			require.Equal(t, tt.expectedFormat, format)
		})
	}
}
