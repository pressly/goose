package column

import (
	"net"

	"github.com/ClickHouse/clickhouse-go/lib/binary"
)

type IPv4 struct {
	base
}

func (*IPv4) Read(decoder *binary.Decoder) (interface{}, error) {
	v, err := decoder.Fixed(4)
	if err != nil {
		return nil, err
	}
	return net.IPv4(v[3], v[2], v[1], v[0]), nil
}

func (ip *IPv4) Write(encoder *binary.Encoder, v interface{}) error {
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
	ip4 := netIP.To4()
	if ip4 == nil {
		return &ErrUnexpectedType{
			T:      v,
			Column: ip,
		}
	}
	if _, err := encoder.Write([]byte{ip4[3], ip4[2], ip4[1], ip4[0]}); err != nil {
		return err
	}
	return nil
}
