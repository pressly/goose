package column

import (
	"net"

	"github.com/ClickHouse/clickhouse-go/lib/binary"
)

type IPv6 struct {
	base
}

func (*IPv6) Read(decoder *binary.Decoder) (interface{}, error) {
	v, err := decoder.Fixed(16)
	if err != nil {
		return nil, err
	}
	return net.IP(v), nil
}

func (ip *IPv6) Write(encoder *binary.Encoder, v interface{}) error {
	var netIP net.IP
	switch v.(type) {
	case string:
		netIP = net.ParseIP(v.(string))
	case net.IP:
		netIP = v.(net.IP)
	case *net.IP:
		netIP = *(v.(*net.IP))
	default:
		return &ErrUnexpectedType{
			T:      v,
			Column: ip,
		}
	}

	if netIP == nil {
		return &ErrUnexpectedType{
			T:      v,
			Column: ip,
		}
	}
	if _, err := encoder.Write([]byte(netIP.To16())); err != nil {
		return err
	}
	return nil
}
