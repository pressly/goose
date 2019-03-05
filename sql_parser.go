package goose

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

type parserState int

const (
	start parserState = iota
	gooseUp
	gooseStatementBeginUp
	gooseStatementEndUp
	gooseDown
	gooseStatementBeginDown
	gooseStatementEndDown
)

const scanBufSize = 4 * 1024 * 1024

var matchEmptyLines = regexp.MustCompile(`^\s*$`)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, scanBufSize)
	},
}

// Split given SQL script into individual statements and return
// SQL statements for given direction (up=true, down=false).
//
// The base case is to simply split on semicolons, as these
// naturally terminate a statement.
//
// However, more complex cases like pl/pgsql can have semicolons
// within a statement. For these cases, we provide the explicit annotations
// 'StatementBegin' and 'StatementEnd' to allow the script to
// tell us to ignore semicolons.
func parseSQLMigration(r io.Reader, direction bool) (stmts []string, useTx bool, err error) {
	var buf bytes.Buffer
	scanBuf := bufferPool.Get().([]byte)
	defer bufferPool.Put(scanBuf)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(scanBuf, scanBufSize)

	stateMachine := start
	useTx = true

	for scanner.Scan() {
		line := scanner.Text()

		const goosePrefix = "-- +goose "
		if strings.HasPrefix(line, goosePrefix) {
			cmd := strings.TrimSpace(line[len(goosePrefix):])

			switch cmd {
			case "Up":
				switch stateMachine {
				case start:
					stateMachine = gooseUp
				default:
					return nil, false, errors.New("failed to parse SQL migration: must start with '-- +goose Up' annotation, see https://github.com/pressly/goose#sql-migrations")
				}

			case "Down":
				switch stateMachine {
				case gooseUp, gooseStatementBeginUp:
					stateMachine = gooseDown
				default:
					return nil, false, errors.New("failed to parse SQL migration: must start with '-- +goose Up' annotation, see https://github.com/pressly/goose#sql-migrations")
				}

			case "StatementBegin":
				switch stateMachine {
				case gooseUp:
					stateMachine = gooseStatementBeginUp
				case gooseDown:
					stateMachine = gooseStatementBeginDown
				default:
					return nil, false, errors.New("failed to parse SQL migration: '-- +goose StatementBegin' must be defined after '-- +goose Up' or '-- +goose Down' annotation, see https://github.com/pressly/goose#sql-migrations")
				}

			case "StatementEnd":
				switch stateMachine {
				case gooseStatementBeginUp:
					stateMachine = gooseStatementEndUp
				case gooseStatementBeginDown:
					stateMachine = gooseStatementEndDown
				default:
					return nil, false, errors.New("failed to parse SQL migration: '-- +goose StatementEnd' must be defined after '-- +goose StatementBegin', see https://github.com/pressly/goose#sql-migrations")
				}

			case "NO TRANSACTION":
				useTx = false

			default:
				return nil, false, errors.Errorf("unknown annotation %q", cmd)
			}
		}

		// Ignore comments.
		if strings.HasPrefix(line, `--`) {
			continue
		}
		// Ignore empty lines.
		if matchEmptyLines.MatchString(line) {
			continue
		}

		// Write SQL line to a buffer.
		if _, err := buf.WriteString(line + "\n"); err != nil {
			return nil, false, errors.Wrap(err, "failed to write to buf")
		}

		// Read SQL body one by line, if we're in the right direction.
		//
		// 1) basic query with semicolon; 2) psql statement
		//
		// Export statement once we hit end of statement.
		switch stateMachine {
		case gooseUp:
			if !direction /*down*/ {
				buf.Reset()
				break
			}
			if endsWithSemicolon(line) {
				stmts = append(stmts, buf.String())
				buf.Reset()
			}
		case gooseDown:
			if direction /*up*/ {
				buf.Reset()
				break
			}
			if endsWithSemicolon(line) {
				stmts = append(stmts, buf.String())
				buf.Reset()
			}
		case gooseStatementEndUp:
			if !direction /*down*/ {
				buf.Reset()
				break
			}
			stmts = append(stmts, buf.String())
			buf.Reset()
		case gooseStatementEndDown:
			if direction /*up*/ {
				buf.Reset()
				break
			}
			stmts = append(stmts, buf.String())
			buf.Reset()
		default:
			return nil, false, errors.New("failed to parse migration: unexpected state %q, see https://github.com/pressly/goose#sql-migrations")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, false, errors.Wrap(err, "failed to scan migration")
	}
	// EOF

	switch stateMachine {
	case start:
		return nil, false, errors.New("failed to parse migration: must start with '-- +goose Up' annotation, see https://github.com/pressly/goose#sql-migrations")
	case gooseStatementBeginUp, gooseStatementBeginDown:
		return nil, false, errors.New("failed to parse migration: missing '-- +goose StatementEnd' annotation")
	}

	if bufferRemaining := strings.TrimSpace(buf.String()); len(bufferRemaining) > 0 {
		return nil, false, errors.Errorf("failed to parse migration: state %q, direction: %v: unexpected unfinished SQL query: %q: missing semicolon?", stateMachine, direction, bufferRemaining)
	}

	return stmts, useTx, nil
}

// Checks the line to see if the line has a statement-ending semicolon
// or if the line contains a double-dash comment.
func endsWithSemicolon(line string) bool {
	scanBuf := bufferPool.Get().([]byte)
	defer bufferPool.Put(scanBuf)

	prev := ""
	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Buffer(scanBuf, scanBufSize)
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		word := scanner.Text()
		if strings.HasPrefix(word, "--") {
			break
		}
		prev = word
	}

	return strings.HasSuffix(prev, ";")
}
