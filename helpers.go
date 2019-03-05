package goose

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type camelSnakeStateMachine int

const (
	begin         camelSnakeStateMachine = iota // 0
	firstAlphaNum                               // 1
	alphaNum                                    // 2
	delimiter                                   // 3
)

func (s camelSnakeStateMachine) next(r rune) camelSnakeStateMachine {
	switch s {
	case begin:
		if isAlphaNum(r) {
			return firstAlphaNum
		}
	case firstAlphaNum:
		if isAlphaNum(r) {
			return alphaNum
		} else {
			return delimiter
		}
	case alphaNum:
		if !isAlphaNum(r) {
			return delimiter
		}
	case delimiter:
		if isAlphaNum(r) {
			return firstAlphaNum
		} else {
			return begin
		}
	}
	return s
}

func lowerCamelCase(str string) string {
	var b strings.Builder

	stateMachine := begin
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

func isAlphaNum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}
