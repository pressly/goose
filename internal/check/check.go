package check

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func HasError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expecting an error: got nil")
	}
}

func IsError(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("expecting specific error:\ngot: %v\nwant: %v", err, target)

	}
}

func Number(t *testing.T, got, want interface{}) {
	t.Helper()
	gotNumber, err := reflectToInt64(got)
	if err != nil {
		t.Fatal(err)
	}
	wantNumber, err := reflectToInt64(want)
	if err != nil {
		t.Fatal(err)
	}
	if gotNumber != wantNumber {
		t.Fatalf("unexpected number value: got:%d want:%d ", gotNumber, wantNumber)
	}
}

func Equal(t *testing.T, got, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("failed deep equal:\ngot:\t%v\nwant:\t%v\v", got, want)
	}
}

func NumberNotZero(t *testing.T, got interface{}) {
	t.Helper()
	gotNumber, err := reflectToInt64(got)
	if err != nil {
		t.Fatal(err)
	}
	if gotNumber == 0 {
		t.Fatalf("unexpected number value: got:%d want non-zero ", gotNumber)
	}
}

func Bool(t *testing.T, got, want bool) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected boolean value: got:%t want:%t", got, want)
	}
}

func Contains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("failed to find substring:\n%s\n\nin string value:\n%s", got, want)
	}
}

func reflectToInt64(v interface{}) (int64, error) {
	switch typ := v.(type) {
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(typ).Int(), nil
	}
	return 0, fmt.Errorf("invalid number: must be int64 type: got:%T", v)
}
