package goose

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type booler bool

func (b *booler) Scan(src interface{}) error {
	switch x := src.(type) {
	case bool:
		*b = booler(x)
	case int:
		*b = x == 1
	case int64:
		*b = x == 1
	case string:
		v := strings.ToLower(x)
		*b = (v == "y") || (v == "yes") || (v == "t") || (v == "true")
	case []uint8:
		if len(x) != 1 {
			return errors.New("invalid []uint8")
		}
		*b = x[0] == 1
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
