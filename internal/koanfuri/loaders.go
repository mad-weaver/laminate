package koanfuri

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
	"github.com/hashicorp/consul/api"
	"github.com/knadh/koanf/providers/consul"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/providers/vault"

	"gocloud.dev/blob"

	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

func (k *KoanfURI) loadCloud() error {

	bucketURL := fmt.Sprintf("%s://%s", k.uri.Scheme, k.uri.Host)
	if k.uri.RawQuery != "" {
		bucketURL = fmt.Sprintf("%s?%s", bucketURL, k.uri.RawQuery)
	}
	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return fmt.Errorf("failed to open bucket: %w", err)
	}
	defer bucket.Close()

	key := strings.TrimPrefix(k.uri.Path, "/")
	reader, err := bucket.NewReader(ctx, key, nil)
	if err != nil {
		return fmt.Errorf("failed to create reader for %s: %w", key, err)
	}
	defer reader.Close()

	// Read all contents into memory
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read contents from %s: %w", key, err)
	}

	if k.dataFormat == "" {
		k.dataFormat = k.detectFormat(data)
	}

	parser, err := k.getParser()
	if err != nil {
		return fmt.Errorf("failed to get parser: %w", err)
	}

	// Load the byte slice into koanf
	if err := k.konfig.Load(rawbytes.Provider(data), parser); err != nil {
		return fmt.Errorf("failed to parse %s: %w", k.uri.String(), err)
	}

	return nil
}

// loadAppConfig loads configuration from AWS AppConfig using AWS SDK v2
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

	// Initialize AWS SDK v2 configuration with automatic credential detection
	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Create AppConfigData client
	client := appconfigdata.NewFromConfig(cfg)

	// Start a configuration session
	startSessionResp, err := client.StartConfigurationSession(ctx, &appconfigdata.StartConfigurationSessionInput{
		ApplicationIdentifier:          &app,
		EnvironmentIdentifier:          &env,
		ConfigurationProfileIdentifier: &config,
	})
	if err != nil {
		return fmt.Errorf("failed to start AppConfig configuration session: %w", err)
	}

	// Get the latest configuration
	getConfigResp, err := client.GetLatestConfiguration(ctx, &appconfigdata.GetLatestConfigurationInput{
		ConfigurationToken: startSessionResp.InitialConfigurationToken,
	})
	if err != nil {
		return fmt.Errorf("failed to get latest AppConfig configuration: %w", err)
	}

	// Read the configuration data
	data := getConfigResp.Configuration
	if len(data) == 0 {
		return fmt.Errorf("empty configuration received from AppConfig")
	}

	// Determine format if not already set
	if k.dataFormat == "" {
		k.dataFormat = k.detectFormat(data)
	}

	// Get the appropriate parser
	parser, err := k.getParser()
	if err != nil {
		return err
	}

	// Use rawbytes provider to load the data
	if err := k.konfig.Load(rawbytes.Provider(data), parser); err != nil {
		return fmt.Errorf("failed to parse AppConfig data: %w", err)
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
