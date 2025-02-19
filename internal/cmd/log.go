package cmd

import (
	"fmt"
	"log/slog"
	"os"
)

var log *slog.Logger

/* create a logger at the specified loglevel */
func createLogger(lvl string) (*slog.Logger, error) {
	level := slog.LevelError
	err := level.UnmarshalText([]byte(lvl))
	if err != nil {
		return nil, fmt.Errorf("invalid log level, %w", err)
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})), nil
}
