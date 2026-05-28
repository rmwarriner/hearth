package observability

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

// Config controls how the logger is initialised.
type Config struct {
	// Level is the minimum log level: "debug", "info", "warn", "error".
	Level string
	// Format is the output format: "json" or "console".
	Format string
}

// sensitiveKeys is the deny-list of field keys whose values must never appear
// in log output. The hook replaces their values with "[REDACTED]".
var sensitiveKeys = map[string]struct{}{
	"email":        {},
	"name":         {},
	"amount":       {},
	"balance":      {},
	"description":  {},
	"payee":        {},
	"memo":         {},
	"account_name": {},
}

// redactWriter wraps an io.Writer and replaces values for sensitive keys in the
// JSON log line before it is written. This operates at the raw-byte level so it
// works regardless of log level or caller.
type redactWriter struct {
	w io.Writer
}

// Write scans the JSON bytes for `"<sensitiveKey>":"<value>"` patterns and
// replaces the value portion with "[REDACTED]". This is a best-effort approach;
// structured zerolog output is always valid JSON so the pattern is reliable.
func (r redactWriter) Write(p []byte) (n int, err error) {
	line := string(p)
	for key := range sensitiveKeys {
		line = redactField(line, key)
	}
	written, err := r.w.Write([]byte(line))
	if err != nil {
		return written, err
	}
	return len(p), nil
}

// redactField replaces the value of a JSON string field named key with "[REDACTED]".
// It handles the pattern `"key":"<value>"` and processes all occurrences in the line.
func redactField(line, key string) string {
	needle := `"` + key + `":"`
	var result strings.Builder
	pos := 0

	for {
		start := strings.Index(line[pos:], needle)
		if start < 0 {
			result.WriteString(line[pos:])
			break
		}
		absStart := pos + start

		// Write everything up to and including the opening quote of the value.
		result.WriteString(line[pos : absStart+len(needle)])
		result.WriteString("[REDACTED]")

		// Advance past the original value to find the closing quote.
		valueEnd := absStart + len(needle)
		for valueEnd < len(line) {
			if line[valueEnd] == '\\' {
				valueEnd += 2
				continue
			}
			if line[valueEnd] == '"' {
				break
			}
			valueEnd++
		}
		// pos now starts after the original value (the closing quote will be written next).
		pos = valueEnd
	}
	return result.String()
}

// NewLogger creates a zerolog.Logger with the given configuration and the
// PII redaction writer installed. The logger should be stored in a
// context.Context via zerolog's log.Ctx pattern, not as a global variable.
func NewLogger(cfg Config) zerolog.Logger {
	var w io.Writer
	if strings.EqualFold(cfg.Format, "console") {
		w = zerolog.ConsoleWriter{Out: os.Stderr}
	} else {
		w = os.Stderr
	}
	return NewLoggerWithWriter(w, cfg)
}

// NewLoggerWithWriter creates a logger that writes to w. Used in tests to
// capture output without writing to stderr.
func NewLoggerWithWriter(w io.Writer, cfg Config) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	return zerolog.New(redactWriter{w: w}).
		Level(level).
		With().
		Timestamp().
		Logger()
}
