package goose

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"strings"
	"unicode"
	"unicode/utf8"
)

type camelSnakeStateMachine int

const ( //                                           _$$_This is some text, OK?!
	idle          camelSnakeStateMachine = iota // 0 ↑                     ↑   ↑
	firstAlphaNum                               // 1     ↑    ↑  ↑    ↑     ↑
	alphaNum                                    // 2      ↑↑↑  ↑  ↑↑↑  ↑↑↑   ↑
	delimiter                                   // 3         ↑  ↑    ↑    ↑   ↑
)

func (s camelSnakeStateMachine) next(r rune) camelSnakeStateMachine {
	switch s {
	case idle:
		if isAlphaNum(r) {
			return firstAlphaNum
		}
	case firstAlphaNum:
		if isAlphaNum(r) {
			return alphaNum
		}
		return delimiter
	case alphaNum:
		if !isAlphaNum(r) {
			return delimiter
		}
	case delimiter:
		if isAlphaNum(r) {
			return firstAlphaNum
		}
		return idle
	}
	return s
}

func camelCase(str string) string {
	var b strings.Builder

	stateMachine := idle
	for i := 0; i < len(str); {
		r, size := utf8.DecodeRuneInString(str[i:])
		i += size
		stateMachine = stateMachine.next(r)
		switch stateMachine {
		case firstAlphaNum:
			b.WriteRune(unicode.ToUpper(r))
		case alphaNum:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func snakeCase(str string) string {
	var b bytes.Buffer

	stateMachine := idle
	for i := 0; i < len(str); {
		r, size := utf8.DecodeRuneInString(str[i:])
		i += size
		stateMachine = stateMachine.next(r)
		switch stateMachine {
		case firstAlphaNum, alphaNum:
			b.WriteRune(unicode.ToLower(r))
		case delimiter:
			b.WriteByte('_')
		}
	}
	if stateMachine == idle {
		return string(bytes.TrimSuffix(b.Bytes(), []byte{'_'}))
	}
	return b.String()
}

func isAlphaNum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

const advisoryLockIDSalt uint = 1486364155

// GenerateAdvisoryLockId inspired by rails migrations, see https://goo.gl/8o9bCT
func generateAdvisoryLockId(schemaname string) (string, error) { // nolint: golint
	sum := crc32.ChecksumIEEE([]byte(schemaname))
	sum = sum * uint32(advisoryLockIDSalt)
	return fmt.Sprint(sum), nil
}