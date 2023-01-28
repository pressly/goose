package sqlparser

import (
	"bufio"
	"bytes"
	"errors"
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
	state   parserState
	verbose bool
}

func newStateMachine(begin parserState, verbose bool) *stateMachine {
	return &stateMachine{
		state:   begin,
		verbose: verbose,
	}
}

func (m *stateMachine) get() parserState {
	return m.state
}

func (m *stateMachine) set(new parserState) {
	m.print("StateMachine: %v => %v", m.state, new)
	m.state = new
}

const (
	grayColor  = "\033[90m"
	resetColor = "\033[00m"
)

func (m *stateMachine) print(msg string, args ...interface{}) {
	msg = "StateMachine: " + msg
	if m.verbose {
		log.Printf(grayColor+msg+resetColor, args...)
	}
}

const scanBufSize = 4 * 1024 * 1024

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, scanBufSize)
		return &buf
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
func ParseSQLMigration(r io.Reader, direction Direction, verbose bool) (stmts []string, useTx bool, err error) {
	scanBufPtr := bufferPool.Get().(*[]byte)
	scanBuf := *scanBufPtr
	defer bufferPool.Put(scanBufPtr)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(scanBuf, scanBufSize)

	sm := newStateMachine(start, verbose)
	useTx = true

	var buf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if verbose {
			log.Println(line)
		}
		if sm.get() == start && strings.TrimSpace(line) == "" {
			continue
		}
		// TODO(mf): validate annotations to avoid common user errors:
		// https://github.com/pressly/goose/issues/163#issuecomment-501736725
		if strings.HasPrefix(line, "--") {
			cmd := strings.TrimSpace(strings.TrimPrefix(line, "--"))

			switch cmd {
			case "+goose Up":
				switch sm.get() {
				case start:
					sm.set(gooseUp)
				default:
					return nil, false, fmt.Errorf("duplicate '-- +goose Up' annotations; stateMachine=%d, see https://github.com/pressly/goose#sql-migrations", sm.state)
				}
				continue

			case "+goose Down":
				switch sm.get() {
				case gooseUp, gooseStatementEndUp:
					sm.set(gooseDown)
				default:
					return nil, false, fmt.Errorf("must start with '-- +goose Up' annotation, stateMachine=%d, see https://github.com/pressly/goose#sql-migrations", sm.state)
				}
				continue

			case "+goose StatementBegin":
				switch sm.get() {
				case gooseUp, gooseStatementEndUp:
					sm.set(gooseStatementBeginUp)
				case gooseDown, gooseStatementEndDown:
					sm.set(gooseStatementBeginDown)
				default:
					return nil, false, fmt.Errorf("'-- +goose StatementBegin' must be defined after '-- +goose Up' or '-- +goose Down' annotation, stateMachine=%d, see https://github.com/pressly/goose#sql-migrations", sm.state)
				}
				continue

			case "+goose StatementEnd":
				switch sm.get() {
				case gooseStatementBeginUp:
					sm.set(gooseStatementEndUp)
				case gooseStatementBeginDown:
					sm.set(gooseStatementEndDown)
				default:
					return nil, false, errors.New("'-- +goose StatementEnd' must be defined after '-- +goose StatementBegin', see https://github.com/pressly/goose#sql-migrations")
				}

			case "+goose NO TRANSACTION":
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
				sm.print("ignore comment")
				continue
			}
		}
		switch sm.get() {
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
		switch sm.get() {
		case gooseUp, gooseStatementBeginUp, gooseStatementEndUp:
			if direction == DirectionDown {
				buf.Reset()
				sm.print("ignore down")
				continue
			}
		case gooseDown, gooseStatementBeginDown, gooseStatementEndDown:
			if direction == DirectionUp {
				buf.Reset()
				sm.print("ignore up")
				continue
			}
		default:
			return nil, false, fmt.Errorf("failed to parse migration: unexpected state %d on line %q, see https://github.com/pressly/goose#sql-migrations", sm.state, line)
		}

		switch sm.get() {
		case gooseUp:
			if endsWithSemicolon(line) {
				stmts = append(stmts, cleanupStatement(buf.String()))
				buf.Reset()
				sm.print("store simple Up query")
			}
		case gooseDown:
			if endsWithSemicolon(line) {
				stmts = append(stmts, cleanupStatement(buf.String()))
				buf.Reset()
				sm.print("store simple Down query")
			}
		case gooseStatementEndUp:
			stmts = append(stmts, cleanupStatement(buf.String()))
			buf.Reset()
			sm.print("store Up statement")
			sm.set(gooseUp)
		case gooseStatementEndDown:
			stmts = append(stmts, cleanupStatement(buf.String()))
			buf.Reset()
			sm.print("store Down statement")
			sm.set(gooseDown)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, false, fmt.Errorf("failed to scan migration: %w", err)
	}
	// EOF

	switch sm.get() {
	case start:
		return nil, false, errors.New("failed to parse migration: must start with '-- +goose Up' annotation, see https://github.com/pressly/goose#sql-migrations")
	case gooseStatementBeginUp, gooseStatementBeginDown:
		return nil, false, errors.New("failed to parse migration: missing '-- +goose StatementEnd' annotation")
	}

	if bufferRemaining := strings.TrimSpace(buf.String()); len(bufferRemaining) > 0 {
		return nil, false, fmt.Errorf("failed to parse migration: state %d, direction: %v: unexpected unfinished SQL query: %q: missing semicolon?", sm.state, direction, bufferRemaining)
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

// Checks the line to see if the line has a statement-ending semicolon
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
