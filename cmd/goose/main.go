package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/joho/godotenv"
	"github.com/mfridman/xflag"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/migrationstats"
	"github.com/pressly/goose/v3/lock"
)

var (
	DefaultMigrationDir = "."

	flags        = flag.NewFlagSet("goose", flag.ExitOnError)
	dir          = flags.String("dir", DefaultMigrationDir, "directory with migration files, (GOOSE_MIGRATION_DIR env variable supported)")
	table        = flags.String("table", "", "migrations table name")
	verbose      = flags.Bool("v", false, "enable verbose mode")
	help         = flags.Bool("h", false, "print help")
	versionFlag  = flags.Bool("version", false, "print version")
	certfile     = flags.String("certfile", "", "file path to root CA's certificates in pem format (only support on mysql)")
	sequential   = flags.Bool("s", false, "use sequential numbering for new migrations")
	allowMissing = flags.Bool("allow-missing", false, "applies missing (out-of-order) migrations")
	sslcert      = flags.String("ssl-cert", "", "file path to SSL certificates in pem format (only support on mysql)")
	sslkey       = flags.String("ssl-key", "", "file path to SSL key in pem format (only support on mysql)")
	noVersioning = flags.Bool("no-versioning", false, "apply migration commands with no versioning, in file order, from directory pointed to")
	noColor      = flags.Bool("no-color", false, "disable color output (NO_COLOR env variable supported)")
	timeout      = flags.Duration("timeout", 0, "maximum allowed duration for queries to run; e.g., 1h13m")
	envFile      = flags.String("env", "", "load environment variables from file (default .env)")
	lockMode     = flags.String("lock", "", "lock mode: none, session, table (GOOSE_LOCK env variable supported)")
)

var version string

func main() {
	ctx := context.Background()

	flags.Usage = usage

	if err := xflag.ParseToEnd(flags, os.Args[1:]); err != nil {
		log.Fatalf("failed to parse args: %v", err)
		return
	}

	if *versionFlag {
		buildInfo, ok := debug.ReadBuildInfo()
		if version == "" && ok && buildInfo != nil && buildInfo.Main.Version != "" {
			version = buildInfo.Main.Version
		}
		fmt.Printf("goose version: %s\n", strings.TrimSpace(version))
		return
	}

	switch *envFile {
	case "":
		// Best effort to load default .env file
		_ = godotenv.Load()
	case "none":
		// Do not load any env file
	default:
		if err := godotenv.Load(*envFile); err != nil {
			log.Fatalf("failed to load env file: %v", err)
		}
	}
	envConfig := loadEnvConfig()

	if *verbose {
		goose.SetVerbose(true)
	}
	if *sequential {
		goose.SetSequential(true)
	}

	// The order of precedence should be: flag > env variable > default value.
	goose.SetTableName(firstNonEmpty(*table, envConfig.table, goose.DefaultTablename))

	args := flags.Args()

	if *help {
		flags.Usage()
		return
	}

	if len(args) == 0 {
		flags.Usage()
		os.Exit(1)
	}

	// The -dir option has not been set, check whether the env variable is set
	// before defaulting to ".".
	if *dir == DefaultMigrationDir && envConfig.dir != "" {
		*dir = envConfig.dir
	}

	switch args[0] {
	case "init":
		if err := gooseInit(*dir); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	case "create":
		if err := goose.RunContext(ctx, "create", nil, *dir, args[1:]...); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	case "fix":
		if err := goose.RunContext(ctx, "fix", nil, *dir); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	case "env":
		for _, env := range envConfig.listEnvs() {
			fmt.Printf("%s=%q\n", env.Name, env.Value)
		}
		return
	case "validate":
		if err := printValidate(*dir, *verbose); err != nil {
			log.Fatalf("goose validate: %v", err)
		}
		return
	case "beta":
		remain := args[1:]
		if len(remain) == 0 {
			log.Println("goose beta: missing subcommand")
			os.Exit(1)
		}
		switch remain[0] {
		case "drivers":
			printDrivers()
		}
		return
	}

	args = mergeArgs(envConfig, args)
	if len(args) < 3 {
		flags.Usage()
		os.Exit(1)
	}

	driver, dbstring, command := args[0], args[1], args[2]
	db, err := goose.OpenDBWithDriver(driver, normalizeDBString(driver, dbstring, *certfile, *sslcert, *sslkey))
	if err != nil {
		log.Fatalf("-dbstring=%q: %v", dbstring, err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("goose: failed to close DB: %v", err)
		}
	}()

	arguments := []string{}
	if len(args) > 3 {
		arguments = append(arguments, args[3:]...)
	}

	if timeout != nil && *timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	// Determine lock mode: flag takes precedence over env variable
	effectiveLockMode := firstNonEmpty(*lockMode, envConfig.lock)

	// Create locker if lock mode is specified (this will fail fast if driver doesn't support it)
	sessionLocker, locker, err := newLocker(driver, effectiveLockMode)
	if err != nil {
		log.Fatalf("goose: %v", err)
	}

	// If locking is enabled, use the Provider API
	if sessionLocker != nil || locker != nil {
		tableName := firstNonEmpty(*table, envConfig.table, goose.DefaultTablename)
		cfg := &runConfig{
			tableName:    tableName,
			verbose:      *verbose,
			allowMissing: *allowMissing,
			noVersioning: *noVersioning,
		}
		if err := runWithProvider(ctx, db, driver, *dir, command, arguments, sessionLocker, locker, cfg); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	}

	// Use the legacy API when no locking is configured
	options := []goose.OptionsFunc{}
	if *noColor || envConfig.noColor {
		options = append(options, goose.WithNoColor(true))
	}
	if *allowMissing {
		options = append(options, goose.WithAllowMissing())
	}
	if *noVersioning {
		options = append(options, goose.WithNoVersioning())
	}
	if err := goose.RunWithOptionsContext(
		ctx,
		command,
		db,
		*dir,
		arguments,
		options...,
	); err != nil {
		log.Fatalf("goose run: %v", err)
	}
}

func printDrivers() {
	drivers := mergeDrivers(sql.Drivers())
	if len(drivers) == 0 {
		fmt.Println("No drivers found")
		return
	}
	fmt.Println("Available drivers:")
	for _, driver := range drivers {
		fmt.Printf("  %s\n", driver)
	}
}

// mergeDrivers merges drivers with a common prefix into a single line.
func mergeDrivers(drivers []string) []string {
	driverMap := make(map[string][]string)

	for _, driver := range drivers {
		parts := strings.Split(driver, "/")
		if len(parts) > 1 {
			// Merge drivers with a common prefix "/"
			prefix := parts[0]
			driverMap[prefix] = append(driverMap[prefix], driver)
		} else {
			// Add drivers without a prefix directly
			driverMap[driver] = append(driverMap[driver], driver)
		}
	}
	var merged []string
	for _, versions := range driverMap {
		sort.Strings(versions)
		merged = append(merged, strings.Join(versions, ", "))
	}
	sort.Strings(merged)
	return merged
}

func mergeArgs(config *envConfig, args []string) []string {
	if len(args) < 1 {
		return args
	}
	if s := config.driver; s != "" {
		args = append([]string{s}, args...)
	}
	if s := config.dbstring; s != "" {
		args = append([]string{args[0], s}, args[1:]...)
	}
	return args
}

func usage() {
	fmt.Println(usagePrefix)
	flags.PrintDefaults()
	fmt.Println(usageCommands)
}

var (
	usagePrefix = `Usage: goose [OPTIONS] DRIVER DBSTRING COMMAND

or

Set environment key
GOOSE_DRIVER=DRIVER
GOOSE_DBSTRING=DBSTRING

Usage: goose [OPTIONS] COMMAND

Drivers:
    postgres
    mysql
    sqlite3
    mssql
    redshift
    tidb
    clickhouse
    ydb
    turso

Examples:
    goose sqlite3 ./foo.db status
    goose sqlite3 ./foo.db create init sql
    goose sqlite3 ./foo.db create add_some_column sql
    goose sqlite3 ./foo.db create fetch_user_data go
    goose sqlite3 ./foo.db up

    goose postgres "user=postgres dbname=postgres sslmode=disable" status
    goose mysql "user:password@/dbname?parseTime=true" status
    goose redshift "postgres://user:password@qwerty.us-east-1.redshift.amazonaws.com:5439/db" status
    goose tidb "user:password@/dbname?parseTime=true" status
    goose mssql "sqlserver://user:password@dbname:1433?database=master" status
    goose clickhouse "tcp://127.0.0.1:9000" status
    goose ydb "grpcs://localhost:2135/local?go_query_mode=scripting&go_fake_tx=scripting&go_query_bind=declare,numeric" status
    goose turso "libsql://dbname.turso.io?authToken=token" status

    GOOSE_DRIVER=sqlite3 GOOSE_DBSTRING=./foo.db goose status
    GOOSE_DRIVER=sqlite3 GOOSE_DBSTRING=./foo.db goose create init sql
    GOOSE_DRIVER=postgres GOOSE_DBSTRING="user=postgres dbname=postgres sslmode=disable" goose status
    GOOSE_DRIVER=mysql GOOSE_DBSTRING="user:password@/dbname" goose status
    GOOSE_DRIVER=redshift GOOSE_DBSTRING="postgres://user:password@qwerty.us-east-1.redshift.amazonaws.com:5439/db" goose status
    GOOSE_DRIVER=turso GOOSE_DBSTRING="libsql://dbname.turso.io?authToken=token" goose status
    GOOSE_DRIVER=clickhouse GOOSE_DBSTRING="clickhouse://user:password@qwerty.clickhouse.cloud:9440/dbname?secure=true&skip_verify=false" goose status

Locking:
    Use the -lock flag or GOOSE_LOCK environment variable to enable locking.
    This prevents concurrent migrations from running at the same time.

    Lock modes:
        none     No locking (default)
        session  Session-level advisory lock (PostgreSQL only)
        table    Table-based distributed lock (PostgreSQL only)

    Examples:
        goose -lock=session postgres "user=postgres dbname=postgres sslmode=disable" up
        GOOSE_LOCK=table goose postgres "user=postgres dbname=postgres sslmode=disable" up

Options:
`

	usageCommands = `
Commands:
    up                   Migrate the DB to the most recent version available
    up-by-one            Migrate the DB up by 1
    up-to VERSION        Migrate the DB to a specific VERSION
    down                 Roll back the version by 1
    down-to VERSION      Roll back to a specific VERSION
    redo                 Re-run the latest migration
    reset                Roll back all migrations
    status               Dump the migration status for the current DB
    version              Print the current version of the database
    create NAME [sql|go] Creates new migration file with the current timestamp
    fix                  Apply sequential ordering to migrations
    validate             Check migration files without running them
`
)

var sqlMigrationTemplate = template.Must(template.New("goose.sql-migration").Parse(`-- Thank you for giving goose a try!
-- 
-- This file was automatically created running goose init. If you're familiar with goose
-- feel free to remove/rename this file, write some SQL and goose up. Briefly,
-- 
-- Documentation can be found here: https://pressly.github.io/goose
--
-- A single goose .sql file holds both Up and Down migrations.
-- 
-- All goose .sql files are expected to have a -- +goose Up annotation.
-- The -- +goose Down annotation is optional, but recommended, and must come after the Up annotation.
-- 
-- The -- +goose NO TRANSACTION annotation may be added to the top of the file to run statements 
-- outside a transaction. Both Up and Down migrations within this file will be run without a transaction.
-- 
-- More complex statements that have semicolons within them must be annotated with 
-- the -- +goose StatementBegin and -- +goose StatementEnd annotations to be properly recognized.
-- 
-- Use GitHub issues for reporting bugs and requesting features, enjoy!

-- +goose Up
SELECT 'up SQL query';

-- +goose Down
SELECT 'down SQL query';
`))

// initDir will create a directory with an empty SQL migration file.
func gooseInit(dir string) error {
	if dir == "" || dir == DefaultMigrationDir {
		dir = "migrations"
	}
	_, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err == nil, errors.Is(err, fs.ErrExist):
		return fmt.Errorf("directory already exists: %s", dir)
	default:
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return goose.CreateWithTemplate(nil, dir, sqlMigrationTemplate, "initial", "sql")
}

func gatherFilenames(filename string) ([]string, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	var filenames []string
	if stat.IsDir() {
		for _, pattern := range []string{"*.sql", "*.go"} {
			file, err := filepath.Glob(filepath.Join(filename, pattern))
			if err != nil {
				return nil, err
			}
			filenames = append(filenames, file...)
		}
	} else {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	return filenames, nil
}

func printValidate(filename string, verbose bool) error {
	filenames, err := gatherFilenames(filename)
	if err != nil {
		return err
	}
	stats, err := migrationstats.GatherStats(
		migrationstats.NewFileWalker(filenames...),
		false,
	)
	if err != nil {
		return err
	}
	// TODO(mf): we should introduce a --debug flag, which allows printing
	// more internal debug information and leave verbose for additional information.
	if !verbose {
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmtPattern := "%v\t%v\t%v\t%v\t%v\t\n"
	fmt.Fprintf(w, fmtPattern, "Type", "Txn", "Up", "Down", "Name")
	fmt.Fprintf(w, fmtPattern, "────", "───", "──", "────", "────")
	for _, m := range stats {
		txnStr := "✔"
		if !m.Tx {
			txnStr = "✘"
		}
		fmt.Fprintf(w, fmtPattern,
			strings.TrimPrefix(filepath.Ext(m.FileName), "."),
			txnStr,
			m.UpCount,
			m.DownCount,
			filepath.Base(m.FileName),
		)
	}
	return w.Flush()
}

type envConfig struct {
	driver   string
	dbstring string
	dir      string
	table    string
	noColor  bool
	lock     string
}

func loadEnvConfig() *envConfig {
	noColorBool, _ := strconv.ParseBool(envOr("NO_COLOR", "false"))
	return &envConfig{
		driver:   envOr("GOOSE_DRIVER", ""),
		dbstring: envOr("GOOSE_DBSTRING", ""),
		table:    envOr("GOOSE_TABLE", ""),
		dir:      envOr("GOOSE_MIGRATION_DIR", DefaultMigrationDir),
		// https://no-color.org/
		noColor: noColorBool,
		lock:    envOr("GOOSE_LOCK", ""),
	}
}

func (c *envConfig) listEnvs() []envVar {
	return []envVar{
		{Name: "GOOSE_DRIVER", Value: c.driver},
		{Name: "GOOSE_DBSTRING", Value: c.dbstring},
		{Name: "GOOSE_MIGRATION_DIR", Value: c.dir},
		{Name: "GOOSE_TABLE", Value: c.table},
		{Name: "GOOSE_LOCK", Value: c.lock},
		{Name: "NO_COLOR", Value: strconv.FormatBool(c.noColor)},
	}
}

type envVar struct {
	Name  string
	Value string
}

// envOr returns os.Getenv(key) if set, or else default.
func envOr(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		val = def
	}
	return val
}

// firstNonEmpty returns the first non-empty string from the provided input or an empty string if all are empty.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// newLocker creates a locker based on the driver and lock mode. If the lock mode is empty or
// "none", it returns nil. If the driver does not support the requested lock mode, it returns an
// error.
//
// Supported lock modes:
//   - "none" or "": No locking (default)
//   - "session": Session-level advisory locking (PostgreSQL only)
//   - "table": Table-based distributed locking (PostgreSQL only)
func newLocker(driver, mode string) (lock.SessionLocker, lock.Locker, error) {
	if mode == "" || mode == "none" {
		return nil, nil, nil
	}

	switch mode {
	case "session":
		switch driver {
		case "postgres", "pgx", "redshift":
			locker, err := lock.NewPostgresSessionLocker()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create postgres session locker: %w", err)
			}
			return locker, nil, nil
		default:
			return nil, nil, fmt.Errorf("driver %q does not support session locking; only postgres, pgx, and redshift support this feature", driver)
		}
	case "table":
		switch driver {
		case "postgres", "pgx", "redshift":
			locker, err := lock.NewPostgresTableLocker()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create postgres table locker: %w", err)
			}
			return nil, locker, nil
		default:
			return nil, nil, fmt.Errorf("driver %q does not support table locking; only postgres, pgx, and redshift support this feature", driver)
		}
	default:
		return nil, nil, fmt.Errorf("invalid lock mode %q; valid options are: none, session, table", mode)
	}
}

// runWithProvider executes a migration command using the Provider API with optional locking support.
func runWithProvider(
	ctx context.Context,
	db *sql.DB,
	driver string,
	dir string,
	command string,
	args []string,
	sessionLocker lock.SessionLocker,
	locker lock.Locker,
	cfg *runConfig,
) error {
	// Map driver to dialect for the Provider
	var dialect goose.Dialect
	switch driver {
	case "postgres", "pgx":
		dialect = goose.DialectPostgres
	case "mysql", "tidb":
		dialect = goose.DialectMySQL
	case "sqlite3", "sqlite":
		dialect = goose.DialectSQLite3
	case "mssql", "sqlserver", "azuresql":
		dialect = goose.DialectMSSQL
	case "redshift":
		dialect = goose.DialectRedshift
	case "clickhouse":
		dialect = goose.DialectClickHouse
	case "vertica":
		dialect = goose.DialectVertica
	case "ydb":
		dialect = goose.DialectYdB
	case "turso", "libsql":
		dialect = goose.DialectTurso
	case "starrocks":
		dialect = goose.DialectStarrocks
	case "spanner":
		dialect = goose.DialectSpanner
	default:
		return fmt.Errorf("unsupported driver %q for provider", driver)
	}

	// Build provider options
	var opts []goose.ProviderOption

	if cfg.tableName != "" {
		opts = append(opts, goose.WithTableName(cfg.tableName))
	}
	if cfg.verbose {
		opts = append(opts, goose.WithVerbose(true))
	}
	if cfg.allowMissing {
		opts = append(opts, goose.WithAllowOutofOrder(true))
	}
	if cfg.noVersioning {
		opts = append(opts, goose.WithDisableVersioning(true))
	}
	if sessionLocker != nil {
		opts = append(opts, goose.WithSessionLocker(sessionLocker))
	}
	if locker != nil {
		opts = append(opts, goose.WithLocker(locker))
	}

	provider, err := goose.NewProvider(dialect, db, os.DirFS(dir), opts...)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	switch command {
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			return err
		}
		for _, r := range results {
			log.Printf("OK   %s (%s)\n", r.Source.Path, r.Duration.Round(1000*1000))
		}
		return nil

	case "up-by-one":
		result, err := provider.UpByOne(ctx)
		if err != nil {
			if errors.Is(err, goose.ErrNoNextVersion) {
				log.Println("goose: no migrations to run")
				return nil
			}
			return err
		}
		log.Printf("OK   %s (%s)\n", result.Source.Path, result.Duration.Round(1000*1000))
		return nil

	case "up-to":
		if len(args) == 0 {
			return fmt.Errorf("up-to requires a VERSION argument")
		}
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("version must be a number (got %q)", args[0])
		}
		results, err := provider.UpTo(ctx, version)
		if err != nil {
			return err
		}
		for _, r := range results {
			log.Printf("OK   %s (%s)\n", r.Source.Path, r.Duration.Round(1000*1000))
		}
		return nil

	case "down":
		result, err := provider.Down(ctx)
		if err != nil {
			if errors.Is(err, goose.ErrNoNextVersion) {
				log.Println("goose: no migrations to run")
				return nil
			}
			return err
		}
		log.Printf("OK   %s (%s)\n", result.Source.Path, result.Duration.Round(1000*1000))
		return nil

	case "down-to":
		if len(args) == 0 {
			return fmt.Errorf("down-to requires a VERSION argument")
		}
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("version must be a number (got %q)", args[0])
		}
		results, err := provider.DownTo(ctx, version)
		if err != nil {
			return err
		}
		for _, r := range results {
			log.Printf("OK   %s (%s)\n", r.Source.Path, r.Duration.Round(1000*1000))
		}
		return nil

	case "redo":
		// Down then up the latest migration
		downResult, err := provider.Down(ctx)
		if err != nil {
			if errors.Is(err, goose.ErrNoNextVersion) {
				return fmt.Errorf("no migrations to redo")
			}
			return err
		}
		log.Printf("OK   %s (down) (%s)\n", downResult.Source.Path, downResult.Duration.Round(1000*1000))

		upResult, err := provider.UpByOne(ctx)
		if err != nil {
			return err
		}
		log.Printf("OK   %s (up) (%s)\n", upResult.Source.Path, upResult.Duration.Round(1000*1000))
		return nil

	case "reset":
		results, err := provider.DownTo(ctx, 0)
		if err != nil {
			return err
		}
		for _, r := range results {
			log.Printf("OK   %s (%s)\n", r.Source.Path, r.Duration.Round(1000*1000))
		}
		return nil

	case "status":
		results, err := provider.Status(ctx)
		if err != nil {
			return err
		}
		log.Println("    Applied At                  Migration")
		log.Println("    =======================================")
		for _, r := range results {
			var appliedAt string
			if r.State == goose.StatePending {
				appliedAt = "Pending"
			} else {
				appliedAt = r.AppliedAt.Format("Mon Jan 02 15:04:05 2006")
			}
			log.Printf("    %-24s -- %s\n", appliedAt, r.Source.Path)
		}
		return nil

	case "version":
		version, err := provider.GetDBVersion(ctx)
		if err != nil {
			return err
		}
		log.Printf("goose: version %d\n", version)
		return nil

	default:
		return fmt.Errorf("command %q is not supported with locking; use without -lock flag", command)
	}
}

// runConfig holds the configuration for running migrations.
type runConfig struct {
	tableName    string
	verbose      bool
	allowMissing bool
	noVersioning bool
}
