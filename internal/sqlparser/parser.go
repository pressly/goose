package sqlparser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
)

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

func FromBool(b bool) Direction {
	if b {
		return DirectionUp
	}
	return DirectionDown
}

type GooseError string

func (e GooseError) Error() string {
	return string(e)
}

func newGooseError(format string, a ...interface{}) GooseError {
	return GooseError(fmt.Sprintf(format, a...))
}

const (
	gooseDocSQLMigration = "https://github.com/pressly/goose#sql-migrations"

	ErrDuplicateGooseUpFormat = "duplicate '-- +goose Up' annotations; stateMachine=%d, see " +
		gooseDocSQLMigration
	ErrMissingGooseUpFormat = "must start with '-- +goose Up' annotation, stateMachine=%d, see " +
		gooseDocSQLMigration
	ErrInvalidStatementBeginFormat = "'-- +goose StatementBegin' must be defined after " +
		"'-- +goose Up' or '-- +goose Down' annotation, stateMachine=%d, see " + gooseDocSQLMigration
	ErrInvalidStatementEndFormat = "'-- +goose StatementEnd' must be defined after " +
		"'-- +goose StatementBegin', see " + gooseDocSQLMigration
)

const (
	AnnotGooseUp             = "+goose Up"
	AnnotGooseDown           = "+goose Down"
	AnnotGooseStatementBegin = "+goose StatementBegin"
	AnnotGooseStatementEnd   = "+goose StatementEnd"
	AnnotGooseNoTransaction  = "+goose NO TRANSACTION"
)

type parserState int

const (
	start                   parserState = iota // 0
	gooseUp                                    // 1
	gooseStatementBeginUp                      // 2
	gooseStatementEndUp                        // 3
	gooseDown                                  // 4
	gooseStatementBeginDown                    // 5
	gooseStatementEndDown                      // 6
)

type stateMachine struct {
	state      parserState
	verbose    bool
	grayColor  string
	resetColor string
}

func newStateMachine(begin parserState, verbose bool) *stateMachine {
	return &stateMachine{
		state:      begin,
		verbose:    verbose,
		grayColor:  "\033[90m",
		resetColor: "\033[00m",
	}
}

func (s *stateMachine) get() parserState {
	return s.state
}

func (s *stateMachine) set(new parserState) {
	s.logVerbose("set %d => %d", s.state, new)
	s.state = new
}

func (s *stateMachine) logVerbose(msg string, args ...interface{}) {
	msg = "StateMachine: " + msg
	if s.verbose {
		log.Printf(s.grayColor+msg+s.resetColor, args...)
	}
}

const scanBufSize = 4 * 1024 * 1024

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, scanBufSize)
		return &buf
	},
}

// ParseSQLMigration split given SQL script into individual statements and return
// SQL statements for given direction (up=true, down=false).
//
// The base case is to simply split on semicolons, as these
// naturally terminate a statement.
//
// However, more complex cases like pl/pgsql can have semicolons
// within a statement. For these cases, we provide the explicit annotations
// 'StatementBegin' and 'StatementEnd' to allow the script to
// tell us to ignore semicolons.
func ParseSQLMigration(r io.Reader, direction Direction, debug bool) (stmts []string, useTx bool, err error) {
	scanBufPtr := bufferPool.Get().(*[]byte)
	scanBuf := *scanBufPtr
	defer bufferPool.Put(scanBufPtr)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(scanBuf, scanBufSize)

	stateMachine := newStateMachine(start, debug)
	useTx = true

	var buf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if debug {
			log.Println(line)
		}
		if stateMachine.get() == start && strings.TrimSpace(line) == "" {
			continue
		}
		// TODO(mf): validate annotations to avoid common user errors:
		// https://github.com/pressly/goose/issues/163#issuecomment-501736725
		if strings.HasPrefix(line, "--") {
			cmd := strings.TrimSpace(strings.TrimPrefix(line, "--"))

			switch cmd {
			case AnnotGooseUp:
				switch stateMachine.get() {
				case start:
					stateMachine.set(gooseUp)
				default:
					return nil, false, newGooseError(ErrDuplicateGooseUpFormat, stateMachine.state)
				}
				continue

			case AnnotGooseDown:
				switch stateMachine.get() {
				case gooseUp, gooseStatementEndUp:
					stateMachine.set(gooseDown)
				default:
					return nil, false, newGooseError(ErrMissingGooseUpFormat, stateMachine.state)
				}
				continue

			case AnnotGooseStatementBegin:
				switch stateMachine.get() {
				case gooseUp, gooseStatementEndUp:
					stateMachine.set(gooseStatementBeginUp)
				case gooseDown, gooseStatementEndDown:
					stateMachine.set(gooseStatementBeginDown)
				default:
					return nil, false, newGooseError(ErrInvalidStatementBeginFormat, stateMachine.state)
				}
				continue

			case AnnotGooseStatementEnd:
				switch stateMachine.get() {
				case gooseStatementBeginUp:
					stateMachine.set(gooseStatementEndUp)
				case gooseStatementBeginDown:
					stateMachine.set(gooseStatementEndDown)
				default:
					return nil, false, newGooseError(ErrInvalidStatementEndFormat)
				}

			case AnnotGooseNoTransaction:
				useTx = false
				continue
			}
		}
		// Once we've started parsing a statement the buffer is no longer empty,
		// we keep all comments up until the end of the statement (the buffer will be reset).
		// All other comments in the file are ignored.
		if buf.Len() == 0 {
			// This check ensures leading comments and empty lines prior to a statement are ignored.
			if strings.HasPrefix(strings.TrimSpace(line), "--") || line == "" {
				stateMachine.logVerbose("ignore comment")
				continue
			}
		}
		switch stateMachine.get() {
		case gooseStatementEndDown, gooseStatementEndUp:
			// Do not include the "+goose StatementEnd" annotation in the final statement.
		default:
			// Write SQL line to a buffer.
			if _, err := buf.WriteString(line + "\n"); err != nil {
				return nil, false, fmt.Errorf("failed to write to buf: %w", err)
			}
		}
		// Read SQL body one by line, if we're in the right direction.
		//
		// 1) basic query with semicolon; 2) psql statement
		//
		// Export statement once we hit end of statement.
		switch stateMachine.get() {
		case gooseUp, gooseStatementBeginUp, gooseStatementEndUp:
			if direction == DirectionDown {
				buf.Reset()
				stateMachine.logVerbose("ignore down")
				continue
			}
		case gooseDown, gooseStatementBeginDown, gooseStatementEndDown:
			if direction == DirectionUp {
				buf.Reset()
				stateMachine.logVerbose("ignore up")
				continue
			}
		default:
			return nil, false, newGooseError("failed to parse migration: unexpected state %d on line %q, see %s", stateMachine.state, line, gooseDocSQLMigration)
		}

		switch stateMachine.get() {
		case gooseUp:
			if endsWithSemicolon(line) {
				stmts = append(stmts, cleanupStatement(buf.String()))
				buf.Reset()
				stateMachine.logVerbose("store simple Up query")
			}
		case gooseDown:
			if endsWithSemicolon(line) {
				stmts = append(stmts, cleanupStatement(buf.String()))
				buf.Reset()
				stateMachine.logVerbose("store simple Down query")
			}
		case gooseStatementEndUp:
			stmts = append(stmts, cleanupStatement(buf.String()))
			buf.Reset()
			stateMachine.logVerbose("store Up statement")
			stateMachine.set(gooseUp)
		case gooseStatementEndDown:
			stmts = append(stmts, cleanupStatement(buf.String()))
			buf.Reset()
			stateMachine.logVerbose("store Down statement")
			stateMachine.set(gooseDown)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, false, fmt.Errorf("failed to scan migration: %w", err)
	}
	// EOF

	switch stateMachine.get() {
	case start:
		return nil, false, newGooseError("failed to parse migration: must start with '-- %s' annotation, see %s", AnnotGooseUp, gooseDocSQLMigration)
	case gooseStatementBeginUp, gooseStatementBeginDown:
		return nil, false, newGooseError("failed to parse migration: missing '-- %s' annotation", AnnotGooseStatementEnd)
	}

	if bufferRemaining := strings.TrimSpace(buf.String()); len(bufferRemaining) > 0 {
		return nil, false, newGooseError("failed to parse migration: state %d, direction: %v: unexpected unfinished SQL query: %q: missing semicolon?", stateMachine.state, direction, bufferRemaining)
	}

	return stmts, useTx, nil
}

// cleanupStatement attempts to find the last semicolon and trims
// the remaining chars from the input string. This is useful for cleaning
// up a statement containing trailing comments or empty lines.
func cleanupStatement(input string) string {
	if n := strings.LastIndex(input, ";"); n > 0 {
		return input[:n+1]
	}
	return input
}

// endsWithSemicolon checks the line to see if the line has a statement-ending semicolon
// or if the line contains a double-dash comment.
func endsWithSemicolon(line string) bool {
	scanBufPtr := bufferPool.Get().(*[]byte)
	scanBuf := *scanBufPtr
	defer bufferPool.Put(scanBufPtr)

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
