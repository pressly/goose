package goosecli

import (
	"flag"

	"github.com/pressly/goose/v3"
)

const (
	dirFlagName          = "dir"
	dbStringFlagName     = "dbstring"
	tableFlagName        = "table"
	timeoutFlagName      = "timeout"
	certfileFlagName     = "certfile"
	sslCertFlagName      = "ssl-cert"
	sslKeyFlagName       = "ssl-key"
	jsonFlagName         = "json"
	noVersioningFlagName = "no-versioning"
	allowMissingFlagName = "allow-missing"
	sequentialFlagName   = "s"
	typeFlagName         = "type"
)

func dirFlag(f *flag.FlagSet) {
	f.String(dirFlagName, "", "Directory with migration files")
}

func dbStringFlag(f *flag.FlagSet) {
	f.String(dbStringFlagName, "", "Database connection string")
}

func tableFlag(f *flag.FlagSet) {
	f.String(tableFlagName, goose.DefaultTablename, "Goose migration table name")
}

func allowMissingFlag(f *flag.FlagSet) {
	f.Bool(allowMissingFlagName, false, "Applies missing (out-of-order) migrations")
}

func noVersioningFlag(f *flag.FlagSet) {
	f.Bool(noVersioningFlagName, false, "Apply migration commands with no versioning, in file order, from directory pointed to")
}

func jsonFlag(f *flag.FlagSet) {
	f.Bool(jsonFlagName, false, "Output results in JSON format")
}

func timeoutFlag(f *flag.FlagSet) {
	f.Duration(timeoutFlagName, 0, "Maximum allowed duration for queries to run; e.g., 1h13m")
}

// commonConnectionFlags are flags that are required for most goose commands which interact with the
// database and open a connection.
func commonConnectionFlags(f *flag.FlagSet) {
	dirFlag(f)
	dbStringFlag(f)
	tableFlag(f)
	timeoutFlag(f)

	// MySQL flags
	f.String(certfileFlagName, "", "File path to root CA's certificates in pem format (mysql only)")
	f.String(sslCertFlagName, "", "File path to SSL certificates in pem format (mysql only)")
	f.String(sslKeyFlagName, "", "File path to SSL key in pem format (mysql only)")
}
