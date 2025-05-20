package koanfuri

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/hcl"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

// KoanfURI represents a URI-based configuration loader using koanf
type KoanfURI struct {
	konfig     *koanf.Koanf
	uri        *url.URL
	dataFormat string
}

// NewKoanfURI creates a new KoanfURI instance from the given URI string
func NewKoanfURI(uri string) (*KoanfURI, error) {
	// Handle stdin special cases
	if uri == "stdin" || uri == "-" {
		return newKoanfURIFromStdin()
	}

	// If the URI doesn't have a scheme, treat it as a file path
	if !strings.Contains(uri, "://") {
		// Convert to absolute path to handle relative paths correctly
		absPath, err := filepath.Abs(uri)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		uri = "file://" + absPath
	}

	// Parse the URI
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}

	k := &KoanfURI{
		konfig: koanf.New("."),
		uri:    parsedURI,
	}

	// Check for scheme hint
	if strings.Contains(parsedURI.Scheme, "+") {
		parts := strings.SplitN(parsedURI.Scheme, "+", 2)
		parsedURI.Scheme = parts[0]
		k.dataFormat = parts[1]
	}

	// Load the configuration based on the scheme
	if err := k.load(); err != nil {
		return nil, err
	}

	return k, nil
}

// newKoanfURIFromStdin creates a new KoanfURI instance reading from stdin
func newKoanfURIFromStdin() (*KoanfURI, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	k := &KoanfURI{
		konfig: koanf.New("."),
		uri:    &url.URL{Scheme: "stdin"},
	}

	// Parse the data
	if err := k.parseData(data); err != nil {
		return nil, err
	}

	return k, nil
}

// load handles loading configuration from the URI based on its scheme
func (k *KoanfURI) load() error {
	switch k.uri.Scheme {
	case "file":
		return k.loadFile()
	case "http", "https":
		return k.loadHTTP()
	case "s3":
		return k.loadS3()
	case "appconfig":
		return k.loadAppConfig()
	case "vault":
		return k.loadVault()
	case "consul":
		return k.loadConsul()
	case "stdin":
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		return k.parseData(data)
	default:
		return fmt.Errorf("unsupported URI scheme: %s", k.uri.Scheme)
	}
}

// parseData detects the format and parses the configuration data
func (k *KoanfURI) parseData(data []byte) error {
	// If format wasn't hinted in the scheme, try to detect from content
	if k.dataFormat == "" {
		k.dataFormat = k.detectFormat(data)
	}

	// Get the appropriate parser
	parser, err := k.getParser()
	if err != nil {
		return err
	}

	// Use rawbytes provider for raw data
	return k.konfig.Load(rawbytes.Provider(data), parser)
}

// getParser returns the appropriate parser based on the data format
func (k *KoanfURI) getParser() (koanf.Parser, error) {
	if k.dataFormat == "" {
		return nil, fmt.Errorf("unable to detect configuration format")
	}

	switch k.dataFormat {
	case "json":
		return json.Parser(), nil
	case "yaml", "yml":
		return yaml.Parser(), nil
	case "toml":
		return toml.Parser(), nil
	case "hcl":
		return hcl.Parser(true), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", k.dataFormat)
	}
}

// detectFormat attempts to determine the configuration format by trying each parser
func (k *KoanfURI) detectFormat(data []byte) string {
	// First try to detect from file extension if available
	if k.uri.Path != "" {
		ext := strings.ToLower(filepath.Ext(k.uri.Path))
		if ext != "" {
			switch ext[1:] { // Remove the leading dot
			case "json", "yaml", "yml", "toml", "hcl":
				return strings.TrimPrefix(ext, ".")
			}
		}
	}

	// Try each parser in turn
	parsers := []struct {
		format string
		parser koanf.Parser
	}{
		{"json", json.Parser()},
		{"toml", toml.Parser()}, // Try TOML before YAML
		{"yaml", yaml.Parser()},
		{"hcl", hcl.Parser(true)},
	}

	for _, p := range parsers {
		// Create a new koanf instance for each attempt
		tempkoanf := koanf.New(".")

		// Try to parse the data
		if err := tempkoanf.Load(rawbytes.Provider(data), p.parser); err == nil {
			// If parsing succeeds and results in at least one key, use this format
			if len(tempkoanf.Keys()) > 0 {
				return p.format
			}
		}
	}

	return ""
}

// GetKonfig returns the underlying koanf.Koanf instance
func (k *KoanfURI) GetKonfig() *koanf.Koanf {
	return k.konfig
}

// GetDataFormat returns the detected data format
func (k *KoanfURI) GetDataFormat() string {
	return k.dataFormat
}

// GetURI returns the parsed URL
func (k *KoanfURI) GetURI() *url.URL {
	return k.uri
}
