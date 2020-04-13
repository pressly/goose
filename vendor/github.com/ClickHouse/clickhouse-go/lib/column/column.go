package column

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/lib/binary"
)

type Column interface {
	Name() string
	CHType() string
	ScanType() reflect.Type
	Read(*binary.Decoder) (interface{}, error)
	Write(*binary.Encoder, interface{}) error
	defaultValue() interface{}
	Depth() int
}

func Factory(name, chType string, timezone *time.Location) (Column, error) {
	switch chType {
	case "Int8":
		return &Int8{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[int8(0)],
			},
		}, nil
	case "Int16":
		return &Int16{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[int16(0)],
			},
		}, nil
	case "Int32":
		return &Int32{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[int32(0)],
			},
		}, nil
	case "Int64":
		return &Int64{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[int64(0)],
			},
		}, nil
	case "UInt8":
		return &UInt8{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[uint8(0)],
			},
		}, nil
	case "UInt16":
		return &UInt16{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[uint16(0)],
			},
		}, nil
	case "UInt32":
		return &UInt32{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[uint32(0)],
			},
		}, nil
	case "UInt64":
		return &UInt64{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[uint64(0)],
			},
		}, nil
	case "Float32":
		return &Float32{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[float32(0)],
			},
		}, nil
	case "Float64":
		return &Float64{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[float64(0)],
			},
		}, nil
	case "String":
		return &String{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[string("")],
			},
		}, nil
	case "UUID":
		return &UUID{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[string("")],
			},
		}, nil
	case "Date":
		_, offset := time.Unix(0, 0).In(timezone).Zone()
		return &Date{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[time.Time{}],
			},
			Timezone: timezone,
			offset:   int64(offset),
		}, nil
	case "IPv4":
		return &IPv4{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[IPv4{}],
			},
		}, nil
	case "IPv6":
		return &IPv6{
			base: base{
				name:    name,
				chType:  chType,
				valueOf: columnBaseTypes[IPv6{}],
			},
		}, nil
	}
	switch {
	case strings.HasPrefix(chType, "DateTime"):
		return &DateTime{
			base: base{
				name:    name,
				chType:  "DateTime",
				valueOf: columnBaseTypes[time.Time{}],
			},
			Timezone: timezone,
		}, nil
	case strings.HasPrefix(chType, "Array"):
		return parseArray(name, chType, timezone)
	case strings.HasPrefix(chType, "Nullable"):
		return parseNullable(name, chType, timezone)
	case strings.HasPrefix(chType, "FixedString"):
		return parseFixedString(name, chType)
	case strings.HasPrefix(chType, "Enum8"), strings.HasPrefix(chType, "Enum16"):
		return parseEnum(name, chType)
	case strings.HasPrefix(chType, "Decimal"):
		return parseDecimal(name, chType)
	case strings.HasPrefix(chType, "SimpleAggregateFunction"):
		if nestedType, err := getNestedType(chType, "SimpleAggregateFunction"); err != nil {
			return nil, err
		} else {
			return Factory(name, nestedType, timezone)
		}
	}
	return nil, fmt.Errorf("column: unhandled type %v", chType)
}

func getNestedType(chType string, wrapType string) (string, error) {
	prefixLen := len(wrapType) + 1
	suffixLen := 1

	if len(chType) > prefixLen+suffixLen {
		nested := strings.Split(chType[prefixLen:len(chType)-suffixLen], ",")
		if len(nested) == 2 {
			return strings.TrimSpace(nested[1]), nil
		}
	}
	return "", fmt.Errorf("column: invalid %s type (%s)", wrapType, chType)
}
