package slogger

import (
	"fmt"
	"log/slog"
	"strings"
)

type (
	Options struct {
		*slog.HandlerOptions
		SourceMode SourceMode
		Format     FormatMode
		ShowTime   bool
		Color      bool
	}
)

func NewOptions(opts *slog.HandlerOptions) *Options {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	ret := &Options{HandlerOptions: opts}
	opts.ReplaceAttr = NewReplaceAttr(ret, ret.ReplaceAttr)

	return ret
}

func (o *Options) SetLevel(level slog.Level) *Options {
	o.Level = level
	return o
}

func (o *Options) SetSourceMode(mode SourceMode) *Options {
	o.SourceMode = mode
	return o
}

func (o *Options) SetFormat(mode FormatMode) *Options {
	o.Format = mode
	return o
}

func (o *Options) SetShowTime(mode bool) *Options {
	o.ShowTime = mode
	return o
}

func (o *Options) SetColor(mode bool) *Options {
	o.Color = mode
	return o
}

func NewReplaceAttr(opts *Options, callback func(groups []string, a slog.Attr) slog.Attr) func(groups []string, a slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		// handle keys
		switch a.Key {
		case slog.LevelKey:
			level := a.Value.Any().(slog.Level)
			if l, exists := levelNames[level]; exists {
				a.Value = slog.StringValue(l.text)
			} else {
				a.Value = slog.StringValue(level.String())
			}
		case slog.TimeKey:
			if !opts.ShowTime {
				return slog.Attr{}
			}
		case slog.SourceKey:
			switch opts.SourceMode {
			case SourceModeNone:
				return slog.Attr{}
			case SourceModeFile:
				if src, ok := a.Value.Any().(*slog.Source); ok {
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", src.File, src.Line))
				}
			case SourceModeShort:
				if src, ok := a.Value.Any().(*slog.Source); ok {
					fullPath := src.File
					seps := strings.Split(fullPath, "/")
					shortPath := seps[len(seps)-1]
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", shortPath, src.Line))
				}
			}
		}

		if callback != nil {
			a = callback(groups, a)
		}
		return a
	}
}
