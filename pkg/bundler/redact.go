package bundler

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Redactor defines the interface for redaction rules.
type Redactor interface {
	Redact(input []byte) []byte
	RedactString(input string) string
}

type regexRedactor struct {
	patterns []*regexp.Regexp
	mask     string
}

func newRedactor() Redactor {
	// Common patterns for secrets and tokens
	return &regexRedactor{
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(password|token|key|secret)\s*[:=]\s*["']?([^"'\s]+)["']?`),
			// Basic IP address regex (IPv4)
			// regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`), // Disabled by default, too aggressive
		},
		mask: "[REDACTED]",
	}
}

func (r *regexRedactor) Redact(input []byte) []byte {
	s := string(input)
	for _, p := range r.patterns {
		s = p.ReplaceAllString(s, "$1: "+r.mask)
	}
	return []byte(s)
}

func (r *regexRedactor) RedactString(input string) string {
	for _, p := range r.patterns {
		input = p.ReplaceAllString(input, "$1: "+r.mask)
	}
	return input
}

// scrubMap recursively scrubs sensitive keys from a map[string]interface{}.
// This is more robust for structured JSON/YAML data than regex replacement.
func scrubMap(data map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	sensitiveKeys := []string{"password", "token", "key", "secret", "authorization"}

	for k, v := range data {
		isSensitive := false
		lowerK := strings.ToLower(k)
		for _, sk := range sensitiveKeys {
			if strings.Contains(lowerK, sk) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			out[k] = "[REDACTED]"
			continue
		}

		switch val := v.(type) {
		case map[string]interface{}:
			out[k] = scrubMap(val)
		case []interface{}:
			out[k] = scrubSlice(val)
		default:
			out[k] = v
		}
	}
	return out
}

func scrubSlice(data []interface{}) []interface{} {
	out := make([]interface{}, len(data))
	for i, v := range data {
		switch val := v.(type) {
		case map[string]interface{}:
			out[i] = scrubMap(val)
		case []interface{}:
			out[i] = scrubSlice(val)
		default:
			out[i] = v
		}
	}
	return out
}

// scrubJSON wraps the map redaction for generic input.
func scrubJSON(input interface{}) (interface{}, error) {
	// Round-trip to scrub if needed, or implement direct traversal
	// For simplicity, we assume input is already a map or slice if we want deep scrubbing.
	// If it's a struct, we should marshal it first then unmarshal to map to scrub generically without reflection complexity.
	var buf []byte
	var err error
	buf, err = json.Marshal(input)
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal(buf, &data); err != nil {
		return nil, err
	}

	switch v := data.(type) {
	case map[string]interface{}:
		return scrubMap(v), nil
	case []interface{}:
		return scrubSlice(v), nil
	}
	return data, nil
}
