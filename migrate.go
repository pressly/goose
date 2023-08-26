package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"
)

var (
	// ErrNoMigrationFiles when no migration files have been found.
	ErrNoMigrationFiles = errors.New("no migration files found")
	// ErrNoCurrentVersion when a current migration version is not found.
	ErrNoCurrentVersion = errors.New("no current version found")
	// ErrNoNextVersion when the next migration version is not found.
	ErrNoNextVersion = errors.New("no next version found")
	// MaxVersion is the maximum allowed version.
	MaxVersion int64 = math.MaxInt64

	registeredGoMigrations = map[int64]*Migration{}
)

// Migrations slice.
type Migrations []*Migration

// helpers so we can use pkg sort
func (ms Migrations) Len() int      { return len(ms) }
func (ms Migrations) Swap(i, j int) { ms[i], ms[j] = ms[j], ms[i] }
func (ms Migrations) Less(i, j int) bool {
	if ms[i].Version == ms[j].Version {
		panic(fmt.Sprintf("goose: duplicate version %v detected:\n%v\n%v", ms[i].Version, ms[i].Source, ms[j].Source))
	}
	return ms[i].Version < ms[j].Version
}

// Current gets the current migration.
func (ms Migrations) Current(current int64) (*Migration, error) {
	for i, migration := range ms {
		if migration.Version == current {
			return ms[i], nil
		}
	}

	return nil, ErrNoCurrentVersion
}

// Next gets the next migration.
func (ms Migrations) Next(current int64) (*Migration, error) {
	for i, migration := range ms {
		if migration.Version > current {
			return ms[i], nil
		}
	}

	return nil, ErrNoNextVersion
}

// Previous : Get the previous migration.
func (ms Migrations) Previous(current int64) (*Migration, error) {
	for i := len(ms) - 1; i >= 0; i-- {
		if ms[i].Version < current {
			return ms[i], nil
		}
	}

	return nil, ErrNoNextVersion
}

// Last gets the last migration.
func (ms Migrations) Last() (*Migration, error) {
	if len(ms) == 0 {
		return nil, ErrNoNextVersion
	}

	return ms[len(ms)-1], nil
}

// Versioned gets versioned migrations.
func (ms Migrations) versioned() (Migrations, error) {
	var migrations Migrations

	// assume that the user will never have more than 19700101000000 migrations
	for _, m := range ms {
		// parse version as timestamp
		versionTime, err := time.Parse(timestampFormat, fmt.Sprintf("%d", m.Version))

		if versionTime.Before(time.Unix(0, 0)) || err != nil {
			migrations = append(migrations, m)
		}
	}

	return migrations, nil
}

// Timestamped gets the timestamped migrations.
func (ms Migrations) timestamped() (Migrations, error) {
	var migrations Migrations

	// assume that the user will never have more than 19700101000000 migrations
	for _, m := range ms {
		// parse version as timestamp
		versionTime, err := time.Parse(timestampFormat, fmt.Sprintf("%d", m.Version))
		if err != nil {
			// probably not a timestamp
			continue
		}

		if versionTime.After(time.Unix(0, 0)) {
			migrations = append(migrations, m)
		}
	}
	return migrations, nil
}

func (ms Migrations) String() string {
	str := ""
	for _, m := range ms {
		str += fmt.Sprintln(m)
	}
	return str
}

// GoMigration is a Go migration func that is run within a transaction.
type GoMigration func(tx *sql.Tx) error

// GoMigrationContext is a Go migration func that is run within a transaction and receives a context.
type GoMigrationContext func(ctx context.Context, tx *sql.Tx) error

// GoMigrationNoTx is a Go migration func that is run outside a transaction.
type GoMigrationNoTx func(db *sql.DB) error

// GoMigrationNoTxContext is a Go migration func that is run outside a transaction and receives a context.
type GoMigrationNoTxContext func(ctx context.Context, db *sql.DB) error

// AddMigration adds Go migrations.
//
// Deprecated: Use AddMigrationContext.
func AddMigration(up, down GoMigration) {
	_, filename, _, _ := runtime.Caller(1)
	AddNamedMigrationContext(filename, withContext(up), withContext(down))
}

// AddMigrationContext adds Go migrations.
func AddMigrationContext(up, down GoMigrationContext) {
	_, filename, _, _ := runtime.Caller(1)
	AddNamedMigrationContext(filename, up, down)
}

// AddNamedMigration adds named Go migrations.
//
// Deprecated: Use AddNamedMigrationContext.
func AddNamedMigration(filename string, up, down GoMigration) {
	AddNamedMigrationContext(filename, withContext(up), withContext(down))
}

// AddNamedMigrationContext adds named Go migrations.
func AddNamedMigrationContext(filename string, up, down GoMigrationContext) {
	if err := register(filename, true, up, down, nil, nil); err != nil {
		panic(err)
	}
}

// AddMigrationNoTx adds Go migrations that will be run outside transaction.
//
// Deprecated: Use AddNamedMigrationNoTxContext.
func AddMigrationNoTx(up, down GoMigrationNoTx) {
	_, filename, _, _ := runtime.Caller(1)
	AddNamedMigrationNoTxContext(filename, withContext(up), withContext(down))
}

// AddMigrationNoTxContext adds Go migrations that will be run outside transaction.
func AddMigrationNoTxContext(up, down GoMigrationNoTxContext) {
	_, filename, _, _ := runtime.Caller(1)
	AddNamedMigrationNoTxContext(filename, up, down)
}

// AddNamedMigrationNoTx adds named Go migrations that will be run outside transaction.
//
// Deprecated: Use AddNamedMigrationNoTxContext.
func AddNamedMigrationNoTx(filename string, up, down GoMigrationNoTx) {
	AddNamedMigrationNoTxContext(filename, withContext(up), withContext(down))
}

// AddNamedMigrationNoTxContext adds named Go migrations that will be run outside transaction.
func AddNamedMigrationNoTxContext(filename string, up, down GoMigrationNoTxContext) {
	if err := register(filename, false, nil, nil, up, down); err != nil {
		panic(err)
	}
}

func register(
	filename string,
	useTx bool,
	up, down GoMigrationContext,
	upNoTx, downNoTx GoMigrationNoTxContext,
) error {
	// Sanity check caller did not mix tx and non-tx based functions.
	if (up != nil || down != nil) && (upNoTx != nil || downNoTx != nil) {
		return fmt.Errorf("cannot mix tx and non-tx based go migrations functions")
	}
	v, _ := NumericComponent(filename)
	if existing, ok := registeredGoMigrations[v]; ok {
		return fmt.Errorf("failed to add migration %q: version %d conflicts with %q",
			filename,
			v,
			existing.Source,
		)
	}
	// Add to global as a registered migration.
	registeredGoMigrations[v] = &Migration{
		Version:           v,
		Next:              -1,
		Previous:          -1,
		Registered:        true,
		Source:            filename,
		UseTx:             useTx,
		UpFnContext:       up,
		DownFnContext:     down,
		UpFnNoTxContext:   upNoTx,
		DownFnNoTxContext: downNoTx,
		// These are deprecated and will be removed in the future.
		// For backwards compatibility we still save the non-context versions in the struct in case someone is using them.
		// Goose does not use these internally anymore and instead uses the context versions.
		UpFn:       withoutContext(up),
		DownFn:     withoutContext(down),
		UpFnNoTx:   withoutContext(upNoTx),
		DownFnNoTx: withoutContext(downNoTx),
	}
	return nil
}

func collectMigrationsFS(
	fsys fs.FS,
	dirpath string,
	current, target int64,
	registered map[int64]*Migration,
) (Migrations, error) {
	if _, err := fs.Stat(fsys, dirpath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%s directory does not exist", dirpath)
		}
		return nil, err
	}
	var migrations Migrations
	// SQL migration files.
	sqlMigrationFiles, err := fs.Glob(fsys, path.Join(dirpath, "*.sql"))
	if err != nil {
		return nil, err
	}
	for _, file := range sqlMigrationFiles {
		v, err := NumericComponent(file)
		if err != nil {
			return nil, fmt.Errorf("could not parse SQL migration file %q: %w", file, err)
		}
		if versionFilter(v, current, target) {
			migrations = append(migrations, &Migration{
				Version:  v,
				Next:     -1,
				Previous: -1,
				Source:   file,
			})
		}
	}
	// Go migration files.
	goMigrations, err := collectGoMigrations(fsys, dirpath, registered, current, target)
	if err != nil {
		return nil, err
	}
	migrations = append(migrations, goMigrations...)
	if len(migrations) == 0 {
		return nil, ErrNoMigrationFiles
	}
	return sortAndConnectMigrations(migrations), nil
}

// CollectMigrations returns all the valid looking migration scripts in the
// migrations folder and go func registry, and key them by version.
func CollectMigrations(dirpath string, current, target int64) (Migrations, error) {
	return collectMigrationsFS(baseFS, dirpath, current, target, registeredGoMigrations)
}

func sortAndConnectMigrations(migrations Migrations) Migrations {
	sort.Sort(migrations)

	// now that we're sorted in the appropriate direction,
	// populate next and previous for each migration
	for i, m := range migrations {
		prev := int64(-1)
		if i > 0 {
			prev = migrations[i-1].Version
			migrations[i-1].Next = m.Version
		}
		migrations[i].Previous = prev
	}

	return migrations
}

func versionFilter(v, current, target int64) bool {
	if target > current {
		return v > current && v <= target
	}
	if target < current {
		return v <= current && v > target
	}
	return false
}

// EnsureDBVersion retrieves the current version for this DB.
// Create and initialize the DB version table if it doesn't exist.
func EnsureDBVersion(db *sql.DB) (int64, error) {
	ctx := context.Background()
	return EnsureDBVersionContext(ctx, db)
}

// EnsureDBVersionContext retrieves the current version for this DB.
// Create and initialize the DB version table if it doesn't exist.
func EnsureDBVersionContext(ctx context.Context, db *sql.DB) (int64, error) {
	dbMigrations, err := store.ListMigrations(ctx, db, TableName())
	if err != nil {
		return 0, createVersionTable(ctx, db)
	}
	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.
	// The first version we find that has been applied is the current version.
	//
	// TODO(mf): for historic reasons, we continue to use the is_applied column,
	// but at some point we need to deprecate this logic and ideally remove
	// this column.
	//
	// For context, see:
	// https://github.com/pressly/goose/pull/131#pullrequestreview-178409168
	//
	// The dbMigrations list is expected to be ordered by descending ID. But
	// in the future we should be able to query the last record only.
	skipLookup := make(map[int64]struct{})
	for _, m := range dbMigrations {
		// Have we already marked this version to be skipped?
		if _, ok := skipLookup[m.VersionID]; ok {
			continue
		}
		// If version has been applied we are done.
		if m.IsApplied {
			return m.VersionID, nil
		}
		// Latest version of migration has not been applied.
		skipLookup[m.VersionID] = struct{}{}
	}
	return 0, ErrNoNextVersion
}

// createVersionTable creates the db version table and inserts the
// initial 0 value into it.
func createVersionTable(ctx context.Context, db *sql.DB) error {
	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := store.CreateVersionTable(ctx, txn, TableName()); err != nil {
		_ = txn.Rollback()
		return err
	}
	if err := store.InsertVersion(ctx, txn, TableName(), 0); err != nil {
		_ = txn.Rollback()
		return err
	}
	return txn.Commit()
}

// GetDBVersion is an alias for EnsureDBVersion, but returns -1 in error.
func GetDBVersion(db *sql.DB) (int64, error) {
	ctx := context.Background()
	return GetDBVersionContext(ctx, db)
}

// GetDBVersionContext is an alias for EnsureDBVersion, but returns -1 in error.
func GetDBVersionContext(ctx context.Context, db *sql.DB) (int64, error) {
	version, err := EnsureDBVersionContext(ctx, db)
	if err != nil {
		return -1, err
	}

	return version, nil
}

// withContext changes the signature of a function that receives one argument to receive a context and the argument.
func withContext[T any](fn func(T) error) func(context.Context, T) error {
	if fn == nil {
		return nil
	}

	return func(ctx context.Context, t T) error {
		return fn(t)
	}
}

// withoutContext changes the signature of a function that receives a context and one argument to receive only the argument.
// When called the passed context is always context.Background().
func withoutContext[T any](fn func(context.Context, T) error) func(T) error {
	if fn == nil {
		return nil
	}

	return func(t T) error {
		return fn(context.Background(), t)
	}
}

// collectGoMigrations collects Go migrations from the filesystem and merges them with registered
// migrations.
//
// If Go migrations have been registered globally, with [goose.AddNamedMigration...], but there are
// no corresponding .go files in the filesystem, add them to the migrations slice.
//
// If Go migrations have been registered, and there are .go files in the filesystem dirpath, ONLY
// include those in the migrations slices.
//
// Lastly, if there are .go files in the filesystem but they have not been registered, raise an
// error. This is to prevent users from accidentally adding valid looking Go files to the migrations
// folder without registering them.
func collectGoMigrations(
	fsys fs.FS,
	dirpath string,
	registeredGoMigrations map[int64]*Migration,
	current, target int64,
) (Migrations, error) {
	// Sanity check registered migrations have the correct version prefix.
	for _, m := range registeredGoMigrations {
		if _, err := NumericComponent(m.Source); err != nil {
			return nil, fmt.Errorf("could not parse go migration file %s: %w", m.Source, err)
		}
	}
	goFiles, err := fs.Glob(fsys, path.Join(dirpath, "*.go"))
	if err != nil {
		return nil, err
	}
	// If there are no Go files in the filesystem and no registered Go migrations, return early.
	if len(goFiles) == 0 && len(registeredGoMigrations) == 0 {
		return nil, nil
	}
	type source struct {
		fullpath string
		version  int64
	}
	// Find all Go files that have a version prefix and are within the requested range.
	var sources []source
	for _, fullpath := range goFiles {
		v, err := NumericComponent(fullpath)
		if err != nil {
			continue // Skip any files that don't have version prefix.
		}
		if strings.HasSuffix(fullpath, "_test.go") {
			continue // Skip Go test files.
		}
		if versionFilter(v, current, target) {
			sources = append(sources, source{
				fullpath: fullpath,
				version:  v,
			})
		}
	}
	var (
		migrations Migrations
	)
	if len(sources) > 0 {
		for _, s := range sources {
			migration, ok := registeredGoMigrations[s.version]
			if ok {
				migrations = append(migrations, migration)
			} else {
				// TODO(mf): something that bothers me about this implementation is it will be
				// lazily evaluated and the error will only be raised if the user tries to run the
				// migration. It would be better to raise an error much earlier in the process.
				migrations = append(migrations, &Migration{
					Version:    s.version,
					Next:       -1,
					Previous:   -1,
					Source:     s.fullpath,
					Registered: false,
				})
			}
		}
	} else {
		// Some users may register Go migrations manually via AddNamedMigration_ functions but not
		// provide the corresponding .go files in the filesystem. In this case, we include them
		// wholesale in the migrations slice.
		//
		// This is a valid use case because users may want to build a custom binary that only embeds
		// the SQL migration files and some other mechanism for registering Go migrations.
		for _, migration := range registeredGoMigrations {
			v, err := NumericComponent(migration.Source)
			if err != nil {
				return nil, fmt.Errorf("could not parse go migration file %s: %w", migration.Source, err)
			}
			if versionFilter(v, current, target) {
				migrations = append(migrations, migration)
			}
		}
	}
	return migrations, nil
}
