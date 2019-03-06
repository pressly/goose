// +build appengine

package godrv

import (
	"net"
	"time"

	"appengine/cloudsql"
)

func init() {
	SetDialer(func(proto, laddr, raddr, user, dbname string, timeout time.Duration) (net.Conn, error) {
		return cloudsql.Dial(raddr)
	})
}
