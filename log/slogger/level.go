package slogger

import (
	"errors"
	"log/slog"
	"strings"
)

func NewLevelVar(val *string) (*slog.LevelVar, error) {
	levelVar := new(slog.LevelVar)

	if val != nil {
		level, err := TranslateToLogLevel(*val)
		if err != nil {
			return nil, err
		}

		levelVar.Set(level)
	}

	return levelVar, nil
}

func TranslateToLogLevel(val string) (slog.Level, error) {
	switch strings.ToLower(val) {
	case "t", "trc", "trace":
		return LevelTrace, nil
	case "d", "dbg", "debug":
		return LevelDebug, nil
	case "i", "inf", "info":
		return LevelInfo, nil
	case "w", "warn", "warning":
		return LevelWarn, nil
	case "e", "err", "error":
		return LevelError, nil
	case "c", "crit", "critical", "fat", "fatal":
		return LevelFatal, nil
	case "p", "panic":
		return LevelPanic, nil
	}

	return LevelError, errors.New("invalid log level: " + val)
}
