package pgdump

import (
	"bytes"
	"strings"
)

// Annotate transforms pg_dump output into a goose-compatible migration file by adding a
// -- +goose Up header and wrapping statements that contain $$ (function bodies, triggers) with
// -- +goose StatementBegin / -- +goose StatementEnd annotations.
//
// Statements are identified as blocks of non-blank lines separated by blank lines.
func Annotate(data []byte) []byte {
	blocks := splitBlocks(data)
	if len(blocks) == 0 {
		return nil
	}

	var out bytes.Buffer
	out.WriteString("-- +goose Up\n")

	for i, block := range blocks {
		if i > 0 {
			out.WriteByte('\n')
		}
		if strings.Contains(block, "$$") {
			out.WriteString("-- +goose StatementBegin\n")
			out.WriteString(block)
			out.WriteString("\n-- +goose StatementEnd\n")
		} else {
			out.WriteString(block)
			out.WriteByte('\n')
		}
	}

	return out.Bytes()
}

// splitBlocks splits text into statement blocks separated by blank lines. Blank lines inside $$
// quoted blocks (function bodies, triggers) are preserved and do not cause a split.
func splitBlocks(data []byte) []string {
	lines := strings.Split(string(bytes.TrimSpace(data)), "\n")
	var blocks []string
	var current strings.Builder
	inDollarQuote := false

	for _, line := range lines {
		// Track $$ state: each occurrence toggles whether we're inside a dollar-quoted block.
		inDollarQuote = inDollarQuote != (strings.Count(line, "$$")%2 == 1)

		if strings.TrimSpace(line) == "" && !inDollarQuote {
			if current.Len() > 0 {
				blocks = append(blocks, current.String())
				current.Reset()
			}
			continue
		}
		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		blocks = append(blocks, current.String())
	}
	return blocks
}
