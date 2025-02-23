package cmd

import (
	"fmt"
	"log/slog"
	"os"
)

var log *slog.Logger

/* create a logger at the specified loglevel */
func createLogger(lvl, output_type string) (*slog.Logger, error) {
	level := slog.LevelError
	err := level.UnmarshalText([]byte(lvl))
	if err != nil {
		return nil, fmt.Errorf("invalid log level, %w", err)
	}

	if output_type == "json" {
		return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})), nil
	}

	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})), nil
}
