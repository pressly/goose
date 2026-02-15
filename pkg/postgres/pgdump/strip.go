package pgdump

import (
	"bytes"
	"strings"
)

// Strip removes noise from pg_dump output, keeping only meaningful DDL statements.
//
// It removes:
//   - SQL comments (lines starting with --)
//   - SET statements (SET statement_timeout, SET lock_timeout, etc.)
//   - SELECT pg_catalog.set_config(...) calls
//   - COMMENT ON statements (auto-generated extension/object comments)
//   - psql meta-commands (lines starting with \)
//   - The "public." schema prefix from identifiers
//   - Consecutive blank lines (collapsed to a single blank line)
//
// Leading and trailing whitespace is trimmed from the final output with a single trailing newline.
func Strip(raw []byte) []byte {
	var out bytes.Buffer
	prevBlank := false

	for _, line := range bytes.Split(raw, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))

		if shouldStrip(trimmed) {
			continue
		}

		// Remove public schema references.
		line = bytes.ReplaceAll(line, []byte("public."), nil)
		line = bytes.ReplaceAll(line, []byte(" WITH SCHEMA public"), nil)

		blank := trimmed == ""
		if blank && prevBlank {
			continue
		}
		prevBlank = blank

		out.Write(line)
		out.WriteByte('\n')
	}

	result := bytes.TrimSpace(out.Bytes())
	if len(result) == 0 {
		return nil
	}
	// Ensure single trailing newline.
	return append(result, '\n')
}

// shouldStrip reports whether a trimmed line should be removed from the output.
func shouldStrip(trimmed string) bool {
	switch {
	case strings.HasPrefix(trimmed, "--"):
		return true
	case strings.HasPrefix(trimmed, "SET "):
		return true
	case strings.HasPrefix(trimmed, "SELECT pg_catalog."):
		return true
	case strings.HasPrefix(trimmed, "COMMENT ON "):
		return true
	case strings.HasPrefix(trimmed, `\`):
		// psql meta-commands emitted by pg_dump (e.g., \restrict, \unrestrict for search_path).
		return true
	default:
		return false
	}
}
