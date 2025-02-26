package apidocs

import (
	"io"
	"log/slog"
)

func InitLogger(w io.Writer) *slog.Logger {
	tsFmt := "2006-01-02 15:04:05"

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				// Convert time to custom format
				t := a.Value.Time()
				a.Value = slog.StringValue(t.Format(tsFmt))
			}
			return a
		},
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

// DisableLogging temporarily disables slog output
// Usage:
//
// originalLogger := DisableLogging()
// defer RestoreLogging(originalLogger)
func DisableLogging() *slog.Logger {
	originalLogger := slog.Default()
	// Suppress logs
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	return originalLogger
}

// RestoreLogging restores the original logger
func RestoreLogging(originalLogger *slog.Logger) {
	slog.SetDefault(originalLogger)
}
