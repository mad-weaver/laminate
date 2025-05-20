package koanfuri

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/knadh/koanf/providers/appconfig"
	"github.com/knadh/koanf/providers/consul"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/providers/s3"
	"github.com/knadh/koanf/providers/vault"
)

// loadS3 loads configuration from an S3 bucket
func (k *KoanfURI) loadS3() error {
	// Get bucket from host and key from path
	bucket := k.uri.Host
	if bucket == "" {
		return fmt.Errorf("invalid S3 URI format, bucket name required: s3://bucket-name/path/to/object")
	}

	// Remove leading slash from path for S3 key
	objectKey := strings.TrimPrefix(k.uri.Path, "/")
	if objectKey == "" {
		return fmt.Errorf("invalid S3 URI format, object key required: s3://bucket-name/path/to/object")
	}

	// Use S3 provider with credentials from environment variables
	provider := s3.Provider(s3.Config{
		Bucket:    bucket,
		ObjectKey: objectKey,
	})

	// If format wasn't hinted in the scheme, try to detect from content
	if k.dataFormat == "" {
		// Get the data for format detection
		data, err := provider.ReadBytes()
		if err != nil {
			return fmt.Errorf("failed to read S3 object for format detection: %w", err)
		}
		k.dataFormat = k.detectFormat(data)
	}

	// Get the appropriate parser
	parser, err := k.getParser()
	if err != nil {
		return err
	}

	if err := k.konfig.Load(provider, parser); err != nil {
		return fmt.Errorf("failed to load from S3: %w", err)
	}

	return nil
}

// loadAppConfig loads configuration from AWS AppConfig
func (k *KoanfURI) loadAppConfig() error {
	// Get application from host
	app := k.uri.Host
	if app == "" {
		return fmt.Errorf("invalid AppConfig URI format, application name required: appconfig://application/environment/configuration")
	}

	// Parse environment and configuration from path
	parts := strings.Split(strings.TrimPrefix(k.uri.Path, "/"), "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid AppConfig URI format, expected appconfig://application/environment/configuration")
	}
	env, config := parts[0], parts[1]

	// Use AppConfig provider (auth from env vars)
	provider := appconfig.Provider(appconfig.Config{
		Application:   app,
		Environment:   env,
		Configuration: config,
	})

	// If format wasn't hinted in the scheme, try to detect from content
	if k.dataFormat == "" {
		// Get the data for format detection
		data, err := provider.ReadBytes()
		if err != nil {
			return fmt.Errorf("failed to read AppConfig for format detection: %w", err)
		}
		k.dataFormat = k.detectFormat(data)
	}

	// Get the appropriate parser
	parser, err := k.getParser()
	if err != nil {
		return err
	}

	if err := k.konfig.Load(provider, parser); err != nil {
		return fmt.Errorf("failed to load from AppConfig: %w", err)
	}

	return nil
}

// loadVault loads configuration from HashiCorp Vault
func (k *KoanfURI) loadVault() error {
	// Get Vault server from host
	server := k.uri.Host
	if server == "" {
		return fmt.Errorf("invalid Vault URI format, server required: vault://server/path/to/secret")
	}

	// Get secret path from path component
	secretPath := strings.TrimPrefix(k.uri.Path, "/")
	if secretPath == "" {
		return fmt.Errorf("invalid Vault URI format, secret path required: vault://server/path/to/secret")
	}

	// Use Vault provider (auth from env vars)
	provider := vault.Provider(vault.Config{
		Address:   fmt.Sprintf("https://%s", server),
		Token:     os.Getenv("VAULT_TOKEN"),
		Path:      secretPath,
		FlatPaths: false,
		Delim:     ".",
	})

	// If format wasn't hinted in the scheme, try to detect from content
	if k.dataFormat == "" {
		// Get the data for format detection
		data, err := provider.ReadBytes()
		if err != nil {
			return fmt.Errorf("failed to read Vault secret for format detection: %w", err)
		}
		k.dataFormat = k.detectFormat(data)
	}

	// Get the appropriate parser
	parser, err := k.getParser()
	if err != nil {
		return err
	}

	if err := k.konfig.Load(provider, parser); err != nil {
		return fmt.Errorf("failed to load from Vault: %w", err)
	}

	return nil
}

// loadConsul loads configuration from Consul KV store
func (k *KoanfURI) loadConsul() error {
	// Get Consul server from host
	server := k.uri.Host
	if server == "" {
		return fmt.Errorf("invalid Consul URI format, server required: consul://server/path/to/key")
	}

	// Get key path from path component
	keyPath := strings.TrimPrefix(k.uri.Path, "/")
	if keyPath == "" {
		return fmt.Errorf("invalid Consul URI format, key path required: consul://server/path/to/key")
	}

	// Use Consul provider (auth from env vars)
	provider := consul.Provider(consul.Config{
		Key:     keyPath,
		Recurse: false,
		Cfg:     &api.Config{Address: server},
	})

	// If format wasn't hinted in the scheme, try to detect from content
	if k.dataFormat == "" {
		// Get the data for format detection
		data, err := provider.ReadBytes()
		if err != nil {
			return fmt.Errorf("failed to read Consul key for format detection: %w", err)
		}
		k.dataFormat = k.detectFormat(data)
	}

	// Get the appropriate parser
	parser, err := k.getParser()
	if err != nil {
		return err
	}

	if err := k.konfig.Load(provider, parser); err != nil {
		return fmt.Errorf("failed to load from Consul: %w", err)
	}

	return nil
}

// loadHTTP loads configuration from an HTTP(S) URL
func (k *KoanfURI) loadHTTP() error {
	resp, err := http.Get(k.uri.String())
	if err != nil {
		return fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// If format wasn't hinted in the scheme, try to detect from content
	if k.dataFormat == "" {
		k.dataFormat = k.detectFormat(data)
	}

	// Get the appropriate parser
	parser, err := k.getParser()
	if err != nil {
		return err
	}

	// Use rawbytes provider for HTTP content since there's no direct HTTP provider
	if err := k.konfig.Load(rawbytes.Provider(data), parser); err != nil {
		return fmt.Errorf("failed to load HTTP content: %w", err)
	}

	return nil
}

// loadFile loads configuration from a local file
func (k *KoanfURI) loadFile() error {
	path := k.uri.Path
	if k.uri.Host != "" {
		path = filepath.Join(k.uri.Host, path)
	}

	// Create file provider instance
	provider := file.Provider(path)

	// If format wasn't hinted in the scheme, try to detect from file contents
	if k.dataFormat == "" {
		data, err := provider.ReadBytes()
		if err != nil {
			return fmt.Errorf("failed to read file for format detection: %w", err)
		}
		k.dataFormat = k.detectFormat(data)
	}

	// Get the appropriate parser
	parser, err := k.getParser()
	if err != nil {
		return err
	}

	// Use file provider with the detected parser
	if err := k.konfig.Load(provider, parser); err != nil {
		return fmt.Errorf("failed to load file: %w", err)
	}

	return nil
}
