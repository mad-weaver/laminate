package laminate

import (
	"slices"
	"strings"

	kenv "github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	urfave "github.com/mad-weaver/laminate/internal/urfave_provider"
	"github.com/urfave/cli/v2"
)

// ParseCLI is a wrapper function that handles converting the command invocation from CLI and returns it as a koanf object. It will
// load environment variables prefixed with "LAMINATE_" unless they are handled by urfave.
//
// Args:
// ctx -> urfave/cli context
func ParseCLI(ctx *cli.Context) (*koanf.Koanf, error) {
	konfig := koanf.New(".")

	// exclude any envvar handled by urfave, this only handles miscellaneous stuff you might want
	// to put into the base koanf object
	excludedEnvVars := []string{
		"LAMINATE_LOGLEVEL",
	}

	// push environment variables prefixed with LAMINATE_ into koanf object
	if err := konfig.Load(kenv.Provider("LAMINATE_", ".", func(s string) string {
		if slices.Contains(excludedEnvVars, s) {
			return "" // Return an empty string to skip this variable
		}
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "LAMINATE_")), "_", ".", -1)
	}), nil); err != nil {
		return nil, err
	}

	// Push CLI args into koanf object
	forcedInclude := []string{"loglevel", "logformat", "merge-strategy"}
	if err := konfig.Load(urfave.NewUrfaveCliProvider(ctx, konfig, ".", false, forcedInclude), nil); err != nil {
		return nil, err
	}

	return konfig, nil
}
