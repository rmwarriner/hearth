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

// redactField replaces the value of any JSON field named key with `"[REDACTED]"`.
// It handles string, number, boolean, null, object, and array values so that
// sensitive keys cannot leak regardless of the value type logged.
func redactField(line, key string) string {
	needle := `"` + key + `":`
	var result strings.Builder
	pos := 0

	for {
		start := strings.Index(line[pos:], needle)
		if start < 0 {
			result.WriteString(line[pos:])
			break
		}
		absStart := pos + start
		afterColon := absStart + len(needle)

		// Write up to and including the colon; replace value with a redacted string.
		result.WriteString(line[pos:afterColon])
		result.WriteString(`"[REDACTED]"`)

		// Advance pos past the original value.
		pos = skipJSONValue(line, afterColon)
	}
	return result.String()
}

// skipJSONValue advances past the JSON value starting at pos in line.
// Handles strings, numbers, booleans, nulls, objects, and arrays.
func skipJSONValue(line string, pos int) int {
	// Skip optional leading whitespace.
	for pos < len(line) && (line[pos] == ' ' || line[pos] == '\t') {
		pos++
	}
	if pos >= len(line) {
		return pos
	}
	switch line[pos] {
	case '"':
		// JSON string — scan to unescaped closing quote.
		pos++
		for pos < len(line) {
			if line[pos] == '\\' {
				pos += 2
				continue
			}
			if line[pos] == '"' {
				pos++
				break
			}
			pos++
		}
	case '{', '[':
		// Object or array — track nesting depth.
		open := line[pos]
		var close byte
		if open == '{' {
			close = '}'
		} else {
			close = ']'
		}
		depth := 1
		pos++
		for pos < len(line) && depth > 0 {
			switch line[pos] {
			case '"':
				pos++
				for pos < len(line) {
					if line[pos] == '\\' {
						pos += 2
						continue
					}
					if line[pos] == '"' {
						pos++
						break
					}
					pos++
				}
				continue
			case open:
				depth++
			case close:
				depth--
			}
			pos++
		}
	default:
		// Number, boolean (true/false), or null — ends at a structural delimiter.
		for pos < len(line) {
			c := line[pos]
			if c == ',' || c == '}' || c == ']' || c == '\n' || c == ' ' || c == '\t' {
				break
			}
			pos++
		}
	}
	return pos
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
