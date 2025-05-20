package sloghelper

import (
	"io"
	"log/slog"
	"os"

	"github.com/golang-cz/devslog"
	"github.com/knadh/koanf/v2"
)

// SetupLogger takes a koanf object and returns a slog.Logger
// object with logging level and format set by koanf.loglevel and
// koanf.logformat.
func SetupLogger(loglevel string, logfile string, logformat string) *slog.Logger {

	var handlerOpts *slog.HandlerOptions
	switch loglevel {
	case "debug":
		handlerOpts = &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
	case "info":
		handlerOpts = &slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo}
	case "warn":
		handlerOpts = &slog.HandlerOptions{AddSource: true, Level: slog.LevelWarn}
	case "error":
		handlerOpts = &slog.HandlerOptions{AddSource: true, Level: slog.LevelError}
	default:
		handlerOpts = &slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo}
	}

	var output io.Writer
	switch logfile {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		output = os.Stderr
	}

	var logger *slog.Logger
	switch logformat {
	case "json":
		logger = slog.New(slog.NewJSONHandler(output, handlerOpts))
	case "text":
		logger = slog.New(slog.NewTextHandler(output, handlerOpts))
	case "rich":
		opts := &devslog.Options{
			HandlerOptions:    handlerOpts,
			NoColor:           true,
			StringIndentation: true,
			StringerFormatter: true,
			SortKeys:          true,
			NewLineAfterLog:   true,
		}
		logger = slog.New(devslog.NewHandler(output, opts))
	default:
		logger = slog.New(slog.NewTextHandler(output, handlerOpts))
	}

	return logger
}

func SetupLoggerfromKoanf(konfig *koanf.Koanf) *slog.Logger {
	return SetupLogger(konfig.String("loglevel"), konfig.String("logfile"), konfig.String("logformat"))
}
