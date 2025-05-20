package laminate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/knadh/koanf/parsers/hcl"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/v2"
	"github.com/mad-weaver/laminate/internal/koanfuri"
	"github.com/urfave/cli/v2"
)

func DefaultApp(c *cli.Context) error {
	// Get the context from metadata
	ctx := c.App.Metadata["ctx"].(context.Context)

	konfig, err := ParseCLI(c) //reparse the CLI context in case a urfave hook was called and altered it since Before() hook called.
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		slog.Debug("received cancellation signal")
		return nil
	default:
		if err := Run(konfig); err != nil {
			return err
		}
	}

	return nil
}

func Run(konfig *koanf.Koanf) error {
	// Validate required source parameter
	source := konfig.String("source")
	if source == "" {
		return fmt.Errorf("source parameter is required")
	}

	// Create base configuration from source
	k, err := koanfuri.NewKoanfURI(source)
	if err != nil {
		return fmt.Errorf("failed to load source configuration: %w", err)
	}

	// Apply patches in order
	for _, patch := range konfig.Strings("patch") {
		p, err := koanfuri.NewKoanfURI(patch)
		if err != nil {
			return fmt.Errorf("failed to load patch %q: %w", patch, err)
		}

		if err := k.Merge(p, konfig.String("merge-strategy")); err != nil {
			return fmt.Errorf("failed to apply patch %q: %w", patch, err)
		}
	}

	// Determine output format, preferring explicitly specified format over source format
	outputFormat := konfig.String("output-format")
	if outputFormat == "" {
		outputFormat = k.GetDataFormat()
	}

	// Get the appropriate parser for the output format
	var parser koanf.Parser
	switch outputFormat {
	case "json":
		parser = json.Parser()
	case "yaml", "yml":
		parser = yaml.Parser()
	case "toml":
		parser = toml.Parser()
	case "hcl":
		parser = hcl.Parser(true)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	// Marshal the configuration using the selected parser
	data, err := k.GetKonfig().Marshal(parser)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to %s: %w", outputFormat, err)
	}

	if len(data) == 0 {
		return fmt.Errorf("marshaled configuration is empty")
	}

	fmt.Println(string(data))
	return nil
}
