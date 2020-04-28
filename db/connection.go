package db

import (
	"database/sql"
	"fmt"
	"github.com/apex/log"
	"github.com/geniusmonkey/gander/creds"
	"github.com/geniusmonkey/gander/env"
	"github.com/geniusmonkey/gander/migration"
	"github.com/geniusmonkey/gander/project"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"strings"
)

var sqlDb *sql.DB

func Setup(project project.Project, env env.Environment, cred creds.Credentials) {
	var err error
	if err = migration.SetDialect(project.Driver); err != nil {
		log.Fatalf("failed to set dialect %s, %v", project.Driver, err)
	}

	var driver string
	var dsn string
	switch project.Driver {
	case "redshift", "cockroach":
		driver = "postgres"
		dsn = buildPostgres(env, cred)
	case "mysql":
		driver = "mysql"
		dsn = buildMysql(env, cred)
	}

	sqlDb, err = sql.Open(driver, dsn)
	if err != nil {
		log.Fatalf("failed to open connection %s", err)
	}
}

func buildMysql(environment env.Environment, cred creds.Credentials) string {
	sb := new(strings.Builder)

	if cred.Username != "" {
		fmt.Fprintf(sb, "%v:%v@", cred.Username, cred.Password)
	}

	fmt.Fprintf(sb, "%v(%v:%v)/%v?", environment.Protocol, environment.Host, environment.Port, environment.Schema)

	for k, v := range environment.Paramas {
		fmt.Fprintf(sb, "%v=%v&", k, v)
	}

	return sb.String()
}

func buildPostgres(environment env.Environment, cred creds.Credentials) string {
	sb := new(strings.Builder)
	sb.WriteString("postgres://")

	if cred.Username != "" {
		fmt.Fprintf(sb, "%v:%v@", cred.Username, cred.Password)
	}

	fmt.Fprintf(sb, "%v:%v/%v?", environment.Host, environment.Port, environment.Schema)
	for k, v := range environment.Paramas {
		fmt.Fprintf(sb, "%v=%v&", k, v)
	}

	return sb.String()
}

func Close() {
	_ = sqlDb.Close()
}

func Get() *sql.DB {
	return sqlDb
}
