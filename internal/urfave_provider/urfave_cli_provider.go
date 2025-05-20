package urfave_provider

import (
	"slices"

	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v2"
)

var _ koanf.Provider = (*UrfaveCliProvider)(nil)

type UrfaveCliProvider struct {
	ctx                *cli.Context
	delim              string
	force_include      []string
	nest_command_flags bool
	k                  *koanf.Koanf
}

// NewUrfaveCliProvider creates a Koanf provider that takes a urfave cli.context,
// dumps all the flags into a flat map, and then uses koanf's delimiter to unflatten
// into a nested structure inside koanf. This implements koanf.Provider interface so it can be used
// with koanf.Load() like any other provider.
//
// Args:
// ctx -> urfave context to dump out
// k -> koanf instance to check for existence of flags under force_include so they can be skipped(because we don't have to force include a config that's already included)
// delim -> specify the delimiter to use for defining nesting structure for koanf (Usually whatever is passed to koanf.New())
// nest_command_flags -> if true, will put urfave flags under "commandflags" instead of under the root.
// force_include -> a list of flagnames that should be processed even if they're not acted on by urfave app. workaround for default values not counted as set.
func NewUrfaveCliProvider(ctx *cli.Context, k *koanf.Koanf, delim string, nest_command_flags bool, force_include []string) *UrfaveCliProvider {
	return &UrfaveCliProvider{
		ctx:                ctx,
		delim:              delim,
		force_include:      force_include,
		nest_command_flags: nest_command_flags,
		k:                  k,
	}
}

func (p *UrfaveCliProvider) Read() (map[string]interface{}, error) {
	tmpMap := make(map[string]interface{})

	for _, flag := range p.ctx.App.Flags {
		flagName := flag.Names()[0]
		if p.ctx.IsSet(flagName) || (!p.k.Exists(flagName) && slices.Contains(p.force_include, flagName)) {
			if _, ok := flag.(*cli.StringSliceFlag); ok {
				tmpMap[flagName] = p.ctx.StringSlice(flagName)
			} else {
				tmpMap[flagName] = p.ctx.Value(flagName)
			}
		}
	}

	for _, flag := range p.ctx.Command.Flags {
		flagName := flag.Names()[0]
		var keyName string
		if p.nest_command_flags {
			keyName = "commandflags" + p.delim + flagName
		} else {
			keyName = flagName
		}
		if p.ctx.IsSet(flagName) || (!p.k.Exists(flagName) && slices.Contains(p.force_include, flagName)) {
			if _, ok := flag.(*cli.StringSliceFlag); ok {
				tmpMap[keyName] = p.ctx.StringSlice(flagName)
			} else {
				tmpMap[keyName] = p.ctx.Value(flagName)
			}
		}
	}

	return maps.Unflatten(tmpMap, p.delim), nil
}

// This function is not implemented, only exists to satisfy koanf.Provider interface
func (p *UrfaveCliProvider) ReadBytes() ([]byte, error) {
	return nil, nil
}
