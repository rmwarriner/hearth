package observability_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hearth-ledger/hearth/internal/observability"
)

func TestNewLogger_RedactsEmailField(t *testing.T) {
	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug", Format: "json"})
	logger.Info().Str("email", "alice@example.com").Msg("test")
	assert.NotContains(t, buf.String(), "alice@example.com")
	assert.Contains(t, buf.String(), "[REDACTED]")
}

func TestNewLogger_RedactsNameField(t *testing.T) {
	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug", Format: "json"})
	logger.Info().Str("name", "Alice Smith").Msg("test")
	assert.NotContains(t, buf.String(), "Alice Smith")
	assert.Contains(t, buf.String(), "[REDACTED]")
}

func TestNewLogger_RedactsAmountField(t *testing.T) {
	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug", Format: "json"})
	logger.Info().Str("amount", "150.00").Msg("test")
	assert.NotContains(t, buf.String(), "150.00")
	assert.Contains(t, buf.String(), "[REDACTED]")
}

func TestNewLogger_NonSensitiveFieldsPassThrough(t *testing.T) {
	sensitiveKeys := []string{"email", "name", "amount", "balance", "description", "payee", "memo", "account_name"}
	nonSensitive := []struct{ key, value string }{
		{"entry_id", "entry-abc-123"},
		{"household_id", "hh-xyz-456"},
		{"operation", "create_account"},
	}

	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug", Format: "json"})

	for _, kv := range nonSensitive {
		buf.Reset()
		logger.Info().Str(kv.key, kv.value).Msg("test")
		assert.Contains(t, buf.String(), kv.value, "expected %q to pass through unredacted", kv.key)
		// Ensure no sensitive keys accidentally appeared.
		for _, sk := range sensitiveKeys {
			assert.NotContains(t, buf.String(), `"`+sk+`"`, "unexpected sensitive key %q in output", sk)
		}
	}
}

func TestNewLogger_RedactsAllDenyListedKeys(t *testing.T) {
	denyList := map[string]string{
		"email":        "user@example.com",
		"name":         "Bob Jones",
		"amount":       "999.99",
		"balance":      "12345.00",
		"description":  "grocery run",
		"payee":        "Whole Foods",
		"memo":         "personal expense",
		"account_name": "Checking Account",
	}

	for key, value := range denyList {
		t.Run(key, func(t *testing.T) {
			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug", Format: "json"})
			logger.Info().Str(key, value).Msg("test")
			out := buf.String()
			assert.NotContains(t, out, value, "key %q value should be redacted", key)
			assert.True(t, strings.Contains(out, "[REDACTED]"), "expected [REDACTED] in output for key %q", key)
		})
	}
}

func TestNewLogger_RedactsNonStringValueTypes(t *testing.T) {
	t.Run("int_value", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug", Format: "json"})
		logger.Info().Int("amount", 42).Msg("test")
		assert.Contains(t, buf.String(), "[REDACTED]")
	})

	t.Run("float_value", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug", Format: "json"})
		logger.Info().Float64("amount", 99.99).Msg("test")
		out := buf.String()
		assert.NotContains(t, out, "99.99", "float value for sensitive key 'amount' should be redacted")
		assert.Contains(t, out, "[REDACTED]")
	})

	t.Run("bool_value", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug", Format: "json"})
		logger.Info().Bool("balance", true).Msg("test")
		assert.Contains(t, buf.String(), "[REDACTED]")
	})
}
