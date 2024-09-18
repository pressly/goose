//go:build no_mysql
// +build no_mysql

package normalizedsn

func DBString(dsn string) (string, error) {
	return dsn, nil
}
