package goose

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type booler bool

func (b *booler) Scan(src interface{}) error {
	switch x := src.(type) {
	case int:
		*b = x == 1
	case int64:
		*b = x == 1
	case string:
		v := strings.ToLower(x)
		*b = (v == "y") || (v == "yes") || (v == "t") || (v == "true")
	case bool:
		*b = booler(x)
	default:
		return fmt.Errorf("unknown scanner source %T", src)
	}
	return nil
}
func (b booler) Value() (driver.Value, error) {
	if b {
		return "Y", nil
	}
	return "N", nil
}
