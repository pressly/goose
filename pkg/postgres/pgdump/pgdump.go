// Package pgdump provides helpers for building pg_dump commands and post-processing their output.
package pgdump

// DefaultExcludeTables is the list of tables excluded from dumps by default. These are goose
// internal tables that should not be part of the schema output.
var DefaultExcludeTables = []string{
	"goose_db_version",
}

// Args returns the pg_dump command-line arguments for a schema-only dump.
//
// The returned slice always starts with "pg_dump" as the first element, suitable for use with
// exec.Command or dockermanage.ExecOptions.Cmd. Goose internal tables are excluded by default.
func Args(database, user string) []string {
	args := []string{
		"pg_dump",
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"-U", user,
		"-d", database,
	}
	for _, t := range DefaultExcludeTables {
		args = append(args, "--exclude-table="+t)
	}
	return args
}
