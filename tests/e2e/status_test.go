package e2e

import (
	"database/sql"
	"fmt"
	"sort"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
)

func TestDetailedStatus(t *testing.T) {
	t.Parallel()

	db, err := newDockerDB(t)
	check.NoError(t, err)
	goose.SetDialect(*dialect)

	migrationsBefore, err := goose.DetailedStatus(db, migrationsDir)
	check.NoError(t, err)
	// check statuses befor apply
	for _, m := range migrationsBefore {
		check.Equal(t, m.IsApplied, false)
	}

	{
		// Apply all up migrations
		err = goose.Up(db, migrationsDir)
		check.NoError(t, err)
		currentVersion, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migrationsBefore[len(migrationsBefore)-1].Version)
	}

	// after full migration
	migrations1, err := goose.DetailedStatus(db, migrationsDir)
	check.NoError(t, err)

	dbMigrations1 := getDbMigrations(t, db)
	// should have all migrations applied
	check.Equal(t, len(dbMigrations1), len(migrations1))
	compareMigrationsStatuses(t, migrations1, dbMigrations1)

	// number migrations to undo
	n := 3

	{
		// undo migrations
		for i := len(migrationsBefore) - 1; i >= len(migrationsBefore)-n; i-- {
			err := migrationsBefore[i].Down(db)
			check.NoError(t, err)
		}
	}

	migrations2, err := goose.DetailedStatus(db, migrationsDir)
	check.NoError(t, err)
	compareMigrationsStatuses(t, migrations2, getDbMigrations(t, db))

	{
		// redo all migrations
		for i := len(migrationsBefore) - n; i < len(migrationsBefore); i++ {
			err := migrationsBefore[i].Up(db)
			check.NoError(t, err)
		}
	}

	migrations3, err := goose.DetailedStatus(db, migrationsDir)
	check.NoError(t, err)
	compareMigrationsStatuses(t, migrations3, getDbMigrations(t, db))
}

func compareMigrationsStatuses(t *testing.T, m goose.Migrations, d map[int64]goose.MigrationRecord) {
	t.Log(logMigrationsStatus(m, d))
	for _, record := range m {
		if !record.IsApplied {
			continue
		}
		dbM, exists := d[record.Version]

		check.Equal(t, exists, true) // no migration in db
		check.Equal(t, dbM.IsApplied, record.IsApplied)
	}
}

func logMigrationsStatus(m goose.Migrations, d map[int64]goose.MigrationRecord) string {
	s := "\nstatus:\n"

	applied := make([]int64, 0, len(m))
	notApplied := make([]int64, 0, len(m))
	for i := range m {
		if m[i].IsApplied {
			applied = append(applied, m[i].Version)
		} else {
			notApplied = append(notApplied, m[i].Version)
		}
	}
	s += fmt.Sprintf("applied - %v\n", applied)
	s += fmt.Sprintf("NOT applied - %v\n", notApplied)

	s += "db:\n"

	keys := make([]int64, 0, len(d))
	for i := range d {
		keys = append(keys, i)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	applied = make([]int64, 0, len(m))
	notApplied = make([]int64, 0, len(m))
	for _, i := range keys {
		if d[i].IsApplied {
			applied = append(applied, d[i].VersionID)
		} else {
			notApplied = append(notApplied, d[i].VersionID)
		}
	}
	s += fmt.Sprintf("applied - %v\n", applied)
	s += fmt.Sprintf("NOT applied - %v\n", notApplied)

	return s
}

func getDbMigrations(t *testing.T, db *sql.DB) map[int64]goose.MigrationRecord {
	q := fmt.Sprintf("SELECT version_id, tstamp, is_applied FROM %s;", goose.TableName())

	rows, err := db.Query(q)
	check.NoError(t, err)

	defer rows.Close()

	result := make(map[int64]goose.MigrationRecord, 20)

	for rows.Next() {
		var row goose.MigrationRecord
		err = rows.Scan(&row.VersionID, &row.TStamp, &row.IsApplied)
		check.NoError(t, err)
		if row.VersionID == 0 {
			continue
		}

		result[row.VersionID] = row
	}

	check.NoError(t, rows.Err())

	return result
}
