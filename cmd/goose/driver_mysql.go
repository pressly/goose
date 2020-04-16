// +build !no_mysql

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"

	"github.com/go-sql-driver/mysql"
	_ "github.com/ziutek/mymysql/godrv"
)

// normalizeMySQLDSN parses the dsn used with the mysql driver to always have
// the parameter `parseTime` set to true. This allows internal goose logic
// to assume that DATETIME/DATE/TIMESTAMP can be scanned into the time.Time
// type.
func normalizeDBString(driver string, str string, certfile string) string {
	if driver == "mysql" {
		var isTLS = certfile != ""
		if isTLS {
			if err := registerTLSConfig(certfile); err != nil {
				log.Fatalf("goose run: %v", err)
			}
		}
		var err error
		str, err = normalizeMySQLDSN(str, isTLS)
		if err != nil {
			log.Fatalf("failed to normalize MySQL connection string: %v", err)
		}
	}
	return str
}

const tlsConfigKey = "custom"

var tlsReg = regexp.MustCompile(`(\?|&)tls=[^&]*(?:&|$)`)

func normalizeMySQLDSN(dsn string, tls bool) (string, error) {
	// If we are sharing a DSN in a different environment, it may contain a TLS
	// setting key with a value name that is not "custom," so clear it.
	dsn = tlsReg.ReplaceAllString(dsn, `$1`)
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	config.ParseTime = true
	if tls {
		config.TLSConfig = tlsConfigKey
	}
	return config.FormatDSN(), nil
}

func registerTLSConfig(pemfile string) error {
	rootCertPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(pemfile)
	if err != nil {
		return err
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return fmt.Errorf("failed to append PEM: %q", pemfile)
	}
	return mysql.RegisterTLSConfig(tlsConfigKey, &tls.Config{
		RootCAs: rootCertPool,
	})
}
