package main

import (
	"database/sql"
	"fmt"
	"log"
	"path"
	"time"
)

var statusCmd = &Command{
	Name:    "status",
	Usage:   "",
	Summary: "dump the migration status for the current DB",
	Help:    `status extended help here...`,
}

type StatusData struct {
	Source string
	Status string
}

func statusRun(cmd *Command, args ...string) {

	conf, err := MakeDBConf()
	if err != nil {
		log.Fatal(err)
	}

	// collect all migrations
	min := int64(0)
	max := int64((1 << 63) - 1)
	mm, e := collectMigrations(conf.MigrationsDir, min, max)
	if e != nil {
		log.Fatal(e)
	}

	db, e := sql.Open(conf.Driver, conf.OpenStr)
	if e != nil {
		log.Fatal("couldn't open DB:", e)
	}
	defer db.Close()

	// must ensure that the version table exists if we're running on a pristine DB
	if _, e := ensureDBVersion(db); e != nil {
		log.Fatal(e)
	}

	fmt.Printf("goose: status for environment '%v'\n", conf.Env)
	fmt.Println("    Applied At                  Migration")
	fmt.Println("    =======================================")
	for _, m := range mm.Migrations {
		printMigrationStatus(db, m.Version, path.Base(m.Source))
	}
}

func printMigrationStatus(db *sql.DB, version int64, script string) {
	var row MigrationRecord
	q := fmt.Sprintf("SELECT tstamp, is_applied FROM goose_db_version WHERE version_id=%d ORDER BY tstamp DESC LIMIT 1", version)
	e := db.QueryRow(q).Scan(&row.TStamp, &row.IsApplied)

	if e != nil && e != sql.ErrNoRows {
		log.Fatal(e)
	}

	var appliedAt string

	if row.IsApplied {
		appliedAt = row.TStamp.Format(time.ANSIC)
	} else {
		appliedAt = "Pending"
	}

	fmt.Printf("    %-24s -- %v\n", appliedAt, script)
}

func init() {
	statusCmd.Run = statusRun
}
