package column

import (
	"fmt"
	"net"
	"reflect"
	"time"
)

type ErrUnexpectedType struct {
	Column Column
	T      interface{}
}

func (err *ErrUnexpectedType) Error() string {
	return fmt.Sprintf("%s: unexpected type %T", err.Column, err.T)
}

var columnBaseTypes = map[interface{}]reflect.Value{
	int8(0):     reflect.ValueOf(int8(0)),
	int16(0):    reflect.ValueOf(int16(0)),
	int32(0):    reflect.ValueOf(int32(0)),
	int64(0):    reflect.ValueOf(int64(0)),
	uint8(0):    reflect.ValueOf(uint8(0)),
	uint16(0):   reflect.ValueOf(uint16(0)),
	uint32(0):   reflect.ValueOf(uint32(0)),
	uint64(0):   reflect.ValueOf(uint64(0)),
	float32(0):  reflect.ValueOf(float32(0)),
	float64(0):  reflect.ValueOf(float64(0)),
	string(""):  reflect.ValueOf(string("")),
	time.Time{}: reflect.ValueOf(time.Time{}),
	IPv4{}:      reflect.ValueOf(net.IP{}),
	IPv6{}:      reflect.ValueOf(net.IP{}),
}

var arrayBaseTypes = map[interface{}]reflect.Type{
	int8(0):     reflect.ValueOf(int8(0)).Type(),
	int16(0):    reflect.ValueOf(int16(0)).Type(),
	int32(0):    reflect.ValueOf(int32(0)).Type(),
	int64(0):    reflect.ValueOf(int64(0)).Type(),
	uint8(0):    reflect.ValueOf(uint8(0)).Type(),
	uint16(0):   reflect.ValueOf(uint16(0)).Type(),
	uint32(0):   reflect.ValueOf(uint32(0)).Type(),
	uint64(0):   reflect.ValueOf(uint64(0)).Type(),
	float32(0):  reflect.ValueOf(float32(0)).Type(),
	float64(0):  reflect.ValueOf(float64(0)).Type(),
	string(""):  reflect.ValueOf(string("")).Type(),
	time.Time{}: reflect.ValueOf(time.Time{}).Type(),
	IPv4{}:      reflect.ValueOf(net.IP{}).Type(),
	IPv6{}:      reflect.ValueOf(net.IP{}).Type(),
}

type base struct {
	name, chType string
	valueOf      reflect.Value
}

func (base *base) Name() string {
	return base.name
}

func (base *base) CHType() string {
	return base.chType
}

func (base *base) ScanType() reflect.Type {
	return base.valueOf.Type()
}

func (base *base) defaultValue() interface{} {
	return base.valueOf.Interface()
}

func (base *base) String() string {
	return fmt.Sprintf("%s (%s)", base.name, base.chType)
}

func (base *base) Depth() int {
	return 0
}
