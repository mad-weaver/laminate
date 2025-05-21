package laminate

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mad-weaver/laminate/internal/sloghelper"
	"github.com/urfave/cli/v2"
)

// NewApp creates a new CLI application instance
func NewApp() *cli.App {
	app := &cli.App{
		Name:     "laminate",
		Usage:    "A CLI tool for layering structured data over structured data",
		Commands: []*cli.Command{
			// Add commands here
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "source",
				Aliases: []string{"s"},
				Usage:   "Specify source data to patch over use '-' for stdin",
			},
			&cli.StringSliceFlag{
				Name:    "patch",
				Aliases: []string{"p"},
				Usage:   "Apply patch file over source -- can be specified multiple times, use '-' for stdin",
				Value:   cli.NewStringSlice(),
			},
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug logging",
				EnvVars: []string{"LAMINATE_DEBUG"},
			},
			&cli.StringFlag{
				Name:    "loglevel",
				Aliases: []string{"l"},
				Usage:   "Specify log level(debug, info, warn, error)",
				Value:   "info",
			},
			&cli.StringFlag{
				Name:    "logformat",
				Aliases: []string{"f"},
				Usage:   "Specify log format(json, text, rich)",
				Value:   "text",
			},
			&cli.StringFlag{
				Name:    "output-format",
				Aliases: []string{"o"},
				Usage:   "Specify output format(json, yaml, toml)",
				Action: func(c *cli.Context, f string) error {
					if f != "json" && f != "yaml" && f != "toml" {
						return fmt.Errorf("invalid output format: %s", f)
					}
					return nil
				},
			},
			&cli.StringFlag{
				Name:  "merge-strategy",
				Value: "overwrite",
				Usage: "Specify list merge strategy (preserve, overwrite)",
			},
		},
		Before: func(c *cli.Context) error {
			// Create context that listens for interrupt signals
			ctx, stop := signal.NotifyContext(context.Background(),
				syscall.SIGTERM,
				syscall.SIGINT,
				os.Interrupt)

			// Store the stop func and context in metadata
			c.App.Metadata = map[string]interface{}{
				"stop": stop,
				"ctx":  ctx,
			}

			// Parse CLI args into koanf config
			konfig, err := ParseCLI(c)
			if err != nil {
				return err
			}

			// Setup logging using the parsed config
			slog.SetDefault(sloghelper.SetupLoggerfromKoanf(konfig))

			return nil
		},
		After: func(c *cli.Context) error {
			// Clean up signal handling
			if stop, ok := c.App.Metadata["stop"].(func()); ok {
				stop()
			}
			return nil
		},
	}
	app.HideHelpCommand = true
	app.Action = DefaultApp
	return app
}
