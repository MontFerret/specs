package validation

import "strings"

// JSONPointer builds an RFC 6901 JSON Pointer from unescaped reference tokens.
func JSONPointer(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	escaped := make([]string, len(parts))

	for index, part := range parts {
		part = strings.ReplaceAll(part, "~", "~0")
		escaped[index] = strings.ReplaceAll(part, "/", "~1")
	}

	return "/" + strings.Join(escaped, "/")
}
