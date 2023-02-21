//go:build !no_mysql
// +build !no_mysql

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
	_ "github.com/ziutek/mymysql/godrv"
)

// normalizeMySQLDSN parses the dsn used with the mysql driver to always have
// the parameter `parseTime` set to true. This allows internal goose logic
// to assume that DATETIME/DATE/TIMESTAMP can be scanned into the time.Time
// type.
func normalizeDBString(driver string, str string, certfile string, sslcert string, sslkey string) string {
	if driver == "mysql" {
		isTLS := certfile != ""
		if isTLS {
			if err := registerTLSConfig(certfile, sslcert, sslkey); err != nil {
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

func normalizeMySQLDSN(dsn string, tls bool) (string, error) {
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

func registerTLSConfig(pemfile string, sslcert string, sslkey string) error {
	rootCertPool := x509.NewCertPool()
	pem, err := os.ReadFile(pemfile)
	if err != nil {
		return err
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return fmt.Errorf("failed to append PEM: %q", pemfile)
	}

	tlsConfig := &tls.Config{
		RootCAs: rootCertPool,
	}
	if sslcert != "" && sslkey != "" {
		cert, err := tls.LoadX509KeyPair(sslcert, sslkey)
		if err != nil {
			return fmt.Errorf("failed to load x509 keypair: %w", err)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}
	return mysql.RegisterTLSConfig(tlsConfigKey, tlsConfig)
}
