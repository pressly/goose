//MySQL driver for Go database/sql package
package godrv

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/ziutek/mymysql/mysql"
	"github.com/ziutek/mymysql/native"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type conn struct {
	my mysql.Conn
}

type rowsRes struct {
	row         mysql.Row
	my          mysql.Result
	simpleQuery mysql.Stmt
}

func errFilter(err error) error {
	if err == io.ErrUnexpectedEOF {
		return driver.ErrBadConn
	}
	if _, ok := err.(net.Error); ok {
		return driver.ErrBadConn
	}
	return err
}

func join(a []string) string {
	n := 0
	for _, s := range a {
		n += len(s)
	}
	b := make([]byte, n)
	n = 0
	for _, s := range a {
		n += copy(b[n:], s)
	}
	return string(b)
}

func (c conn) parseQuery(query string, args []driver.Value) (string, error) {
	if len(args) == 0 {
		return query, nil
	}
	if strings.ContainsAny(query, `'"`) {
		return "", nil
	}
	q := make([]string, 2*len(args)+1)
	n := 0
	for _, a := range args {
		i := strings.IndexRune(query, '?')
		if i == -1 {
			return "", errors.New("number of parameters doesn't match number of placeholders")
		}
		var s string
		switch v := a.(type) {
		case nil:
			s = "NULL"
		case string:
			s = "'" + c.my.Escape(v) + "'"
		case []byte:
			s = "'" + c.my.Escape(string(v)) + "'"
		case int64:
			s = strconv.FormatInt(v, 10)
		case time.Time:
			s = "'" + v.Format(mysql.TimeFormat) + "'"
		case bool:
			if v {
				s = "1"
			} else {
				s = "0"
			}
		case float64:
			s = strconv.FormatFloat(v, 'e', 12, 64)
		default:
			panic(fmt.Sprintf("%v (%T) can't be handled by godrv"))
		}
		q[n] = query[:i]
		q[n+1] = s
		query = query[i+1:]
		n += 2
	}
	q[n] = query
	return join(q), nil
}

func (c conn) Exec(query string, args []driver.Value) (driver.Result, error) {
	q, err := c.parseQuery(query, args)
	if err != nil {
		return nil, err
	}
	if len(q) == 0 {
		return nil, driver.ErrSkip
	}
	res, err := c.my.Start(q)
	if err != nil {
		return nil, errFilter(err)
	}
	return &rowsRes{my: res}, nil
}

var textQuery = mysql.Stmt(new(native.Stmt))

func (c conn) Query(query string, args []driver.Value) (driver.Rows, error) {
	q, err := c.parseQuery(query, args)
	if err != nil {
		return nil, err
	}
	if len(q) == 0 {
		return nil, driver.ErrSkip
	}
	res, err := c.my.Start(q)
	if err != nil {
		return nil, errFilter(err)
	}
	return &rowsRes{row: res.MakeRow(), my: res, simpleQuery: textQuery}, nil
}

type stmt struct {
	my   mysql.Stmt
	args []interface{}
}

func (s *stmt) run(args []driver.Value) (*rowsRes, error) {
	for i, v := range args {
		s.args[i] = interface{}(v)
	}
	res, err := s.my.Run(s.args...)
	if err != nil {
		return nil, errFilter(err)
	}
	return &rowsRes{my: res}, nil
}

func (c conn) Prepare(query string) (driver.Stmt, error) {
	st, err := c.my.Prepare(query)
	if err != nil {
		return nil, errFilter(err)
	}
	return &stmt{st, make([]interface{}, st.NumParam())}, nil
}

func (c conn) Close() (err error) {
	err = c.my.Close()
	c.my = nil
	if err != nil {
		err = errFilter(err)
	}
	return
}

type tx struct {
	my mysql.Transaction
}

func (c conn) Begin() (driver.Tx, error) {
	t, err := c.my.Begin()
	if err != nil {
		return nil, errFilter(err)
	}
	return tx{t}, nil
}

func (t tx) Commit() (err error) {
	err = t.my.Commit()
	if err != nil {
		err = errFilter(err)
	}
	return
}

func (t tx) Rollback() (err error) {
	err = t.my.Rollback()
	if err != nil {
		err = errFilter(err)
	}
	return
}

func (s *stmt) Close() (err error) {
	err = s.my.Delete()
	s.my = nil
	if err != nil {
		err = errFilter(err)
	}
	return
}

func (s *stmt) NumInput() int {
	return s.my.NumParam()
}

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.run(args)
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	r, err := s.run(args)
	if err != nil {
		return nil, err
	}
	r.row = r.my.MakeRow()
	return r, nil
}

func (r *rowsRes) LastInsertId() (int64, error) {
	return int64(r.my.InsertId()), nil
}

func (r *rowsRes) RowsAffected() (int64, error) {
	return int64(r.my.AffectedRows()), nil
}

func (r *rowsRes) Columns() []string {
	flds := r.my.Fields()
	cls := make([]string, len(flds))
	for i, f := range flds {
		cls[i] = f.Name
	}
	return cls
}

func (r *rowsRes) Close() error {
	if r.my == nil {
		return nil // closed before
	}
	if err := r.my.End(); err != nil {
		return errFilter(err)
	}
	if r.simpleQuery != nil && r.simpleQuery != textQuery {
		if err := r.simpleQuery.Delete(); err != nil {
			return errFilter(err)
		}
	}
	r.my = nil
	return nil
}

// DATE, DATETIME, TIMESTAMP are treated as they are in Local time zone
func (r *rowsRes) Next(dest []driver.Value) error {
	if r.my == nil {
		return io.EOF // closed before
	}
	err := r.my.ScanRow(r.row)
	if err == nil {
		if r.simpleQuery == textQuery {
			// workaround for time.Time from text queries
			for i, f := range r.my.Fields() {
				if r.row[i] != nil {
					switch f.Type {
					case native.MYSQL_TYPE_TIMESTAMP, native.MYSQL_TYPE_DATETIME,
						native.MYSQL_TYPE_DATE, native.MYSQL_TYPE_NEWDATE:
						r.row[i] = r.row.ForceLocaltime(i)
					}
				}
			}
		}
		for i, d := range r.row {
			dest[i] = driver.Value(d)
		}
		return nil
	}
	if err != io.EOF {
		return errFilter(err)
	}
	if r.simpleQuery != nil && r.simpleQuery != textQuery {
		if err = r.simpleQuery.Delete(); err != nil {
			return errFilter(err)
		}
	}
	r.my = nil
	return io.EOF
}

type Driver struct {
	// Defaults
	proto, laddr, raddr, user, passwd, db string
	timeout                               time.Duration
	dialer                                Dialer

	initCmds []string
}

// Open new connection. The uri need to have the following syntax:
//
//   [PROTOCOL_SPECFIIC*]DBNAME/USER/PASSWD
//
// where protocol spercific part may be empty (this means connection to
// local server using default protocol). Currently possible forms:
//
//   DBNAME/USER/PASSWD
//   unix:SOCKPATH*DBNAME/USER/PASSWD
//   unix:SOCKPATH,OPTIONS*DBNAME/USER/PASSWD
//   tcp:ADDR*DBNAME/USER/PASSWD
//   tcp:ADDR,OPTIONS*DBNAME/USER/PASSWD
//   cloudsql:INSTANCE*DBNAME/USER/PASSWD
//
// OPTIONS can contain comma separated list of options in form:
//   opt1=VAL1,opt2=VAL2,boolopt3,boolopt4
// Currently implemented options:
//   laddr   - local address/port (eg. 1.2.3.4:0)
//   timeout - connect timeout in format accepted by time.ParseDuration
func (d *Driver) Open(uri string) (driver.Conn, error) {
	cfg := *d // copy default configuration
	pd := strings.SplitN(uri, "*", 2)
	if len(pd) == 2 {
		// Parse protocol part of URI
		p := strings.SplitN(pd[0], ":", 2)
		if len(p) != 2 {
			return nil, errors.New("Wrong protocol part of URI")
		}
		cfg.proto = p[0]
		options := strings.Split(p[1], ",")
		cfg.raddr = options[0]
		for _, o := range options[1:] {
			kv := strings.SplitN(o, "=", 2)
			var k, v string
			if len(kv) == 2 {
				k, v = kv[0], kv[1]
			} else {
				k, v = o, "true"
			}
			switch k {
			case "laddr":
				cfg.laddr = v
			case "timeout":
				to, err := time.ParseDuration(v)
				if err != nil {
					return nil, err
				}
				cfg.timeout = to
			default:
				return nil, errors.New("Unknown option: " + k)
			}
		}
		// Remove protocol part
		pd = pd[1:]
	}
	// Parse database part of URI
	dup := strings.SplitN(pd[0], "/", 3)
	if len(dup) != 3 {
		return nil, errors.New("Wrong database part of URI")
	}
	cfg.db = dup[0]
	cfg.user = dup[1]
	cfg.passwd = dup[2]

	c := conn{mysql.New(
		cfg.proto, cfg.laddr, cfg.raddr, cfg.user, cfg.passwd, cfg.db,
	)}
	if d.dialer != nil {
		dialer := func(proto, laddr, raddr string, timeout time.Duration) (
			net.Conn, error) {

			return d.dialer(proto, laddr, raddr, cfg.user, cfg.passwd, timeout)
		}
		c.my.SetDialer(dialer)
	}

	// Establish the connection
	c.my.SetTimeout(cfg.timeout)
	for _, q := range cfg.initCmds {
		c.my.Register(q) // Register initialisation commands
	}
	if err := c.my.Connect(); err != nil {
		return nil, errFilter(err)
	}
	c.my.NarrowTypeSet(true)
	c.my.FullFieldInfo(false)
	return &c, nil
}

// Register registers initialisation commands.
// This is workaround, see http://codereview.appspot.com/5706047
func (drv *Driver) Register(query string) {
	drv.initCmds = append(d.initCmds, query)
}

// Dialer can be used to dial connections to MySQL. If Dialer returns (nil, nil)
// the hook is skipped and normal dialing proceeds. user and dbname are there
// only for logging.
type Dialer func(proto, laddr, raddr, user, dbname string, timeout time.Duration) (net.Conn, error)

// SetDialer sets custom Dialer used by Driver to make connections
func (drv *Driver) SetDialer(dialer Dialer) {
	drv.dialer = dialer
}

// Driver automatically registered in database/sql
var d = Driver{proto: "tcp", raddr: "127.0.0.1:3306"}

// Register calls Register method on driver registered in database/sql
func Register(query string) {
	d.Register(query)
}

// SetDialer calls SetDialer method on driver registered in database/sql
func SetDialer(dialer Dialer) {
	d.SetDialer(dialer)
}

func init() {
	Register("SET NAMES utf8")
	sql.Register("mymysql", &d)
}

// Version returns mymysql version string
func Version() string {
	return mysql.Version()
}
