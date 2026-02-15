package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/pkg/dockermanage"
	"github.com/pressly/goose/v3/pkg/dockermanage/dockerpostgres"
	"github.com/pressly/goose/v3/pkg/postgres/pgdump"
)

func runDump(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("goose dump", flag.ContinueOnError)
	f.Usage = dumpUsage

	var (
		dbstring       = f.String("dbstring", "", "PostgreSQL connection string (GOOSE_DBSTRING env var supported)")
		docker         = f.Bool("docker", false, "use Docker to run pg_dump (and optionally Postgres)")
		fromMigrations = f.Bool("from-migrations", false, "spin up ephemeral Postgres, apply migrations, then dump (requires --docker)")
		migrationsDir  = f.String("dir", ".", "migration files directory (GOOSE_MIGRATION_DIR env var supported)")
		strip          = f.Bool("strip", false, "remove comments, SET statements, OWNER TO, and pg_catalog noise")
		gooseAnnotate  = f.Bool("goose", false, "add goose annotations (-- +goose Up, StatementBegin/End for $$ blocks)")
		out            = f.String("out", "", "write output to file instead of stdout")
		pgVersion      = f.String("pg-version", "16", "PostgreSQL major version for Docker image")
	)

	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Env var fallbacks.
	if *dbstring == "" {
		*dbstring = os.Getenv("GOOSE_DBSTRING")
	}
	if *migrationsDir == "." {
		if v := os.Getenv("GOOSE_MIGRATION_DIR"); v != "" {
			*migrationsDir = v
		}
	}

	// Validation.
	if *fromMigrations && !*docker {
		return errors.New("--from-migrations requires --docker")
	}
	if !*fromMigrations && *dbstring == "" {
		return errors.New("--dbstring is required (or set GOOSE_DBSTRING) unless --from-migrations is used")
	}

	var (
		output []byte
		err    error
	)
	switch {
	case *fromMigrations:
		// Case 3: ephemeral Docker Postgres + migrations + pg_dump
		output, err = dumpFromMigrations(ctx, *migrationsDir, *pgVersion)
	case *docker:
		// Case 2: Docker pg_dump against existing DB
		output, err = dumpDockerExisting(ctx, *dbstring, *pgVersion)
	default:
		// Case 1: local pg_dump
		output, err = dumpLocal(ctx, *dbstring)
	}
	if err != nil {
		return err
	}

	if *strip {
		output = pgdump.Strip(output)
	}
	if *gooseAnnotate {
		output = pgdump.Annotate(output)
	}

	if *out != "" {
		if err := os.WriteFile(*out, output, 0644); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		log.Printf("schema written to %s", *out)
		return nil
	}
	_, err = os.Stdout.Write(output)
	return err
}

// dumpLocal runs pg_dump from the local PATH against the given DSN.
func dumpLocal(ctx context.Context, dbstring string) ([]byte, error) {
	pgDumpPath, err := exec.LookPath("pg_dump")
	if err != nil {
		return nil, fmt.Errorf("pg_dump not found in PATH: %w\n\nInstall pg_dump or use --docker to run it inside a container", err)
	}
	args := []string{
		"--schema-only",
		"--no-owner",
		"--no-privileges",
	}
	for _, t := range pgdump.DefaultExcludeTables {
		args = append(args, "--exclude-table="+t)
	}
	args = append(args, dbstring)
	cmd := exec.CommandContext(ctx, pgDumpPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pg_dump failed: %w\n%s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// dumpDockerExisting runs pg_dump inside a Docker container against an existing external DB.
func dumpDockerExisting(ctx context.Context, dbstring string, pgVersion string) ([]byte, error) {
	manager, err := dockermanage.NewManager(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))
	if err != nil {
		return nil, fmt.Errorf("create Docker manager: %w", err)
	}
	defer manager.Close()

	image := fmt.Sprintf("postgres:%s-alpine", pgVersion)

	// Start a temporary container just for the pg_dump binary. We don't need a running Postgres
	// server, but the postgres image has pg_dump included. Use a dummy POSTGRES_PASSWORD to satisfy
	// the image entrypoint.
	container, err := manager.Start(ctx,
		dockermanage.WithImage(image),
		dockermanage.WithContainerPortTCP(5432),
		dockermanage.WithEnv("POSTGRES_PASSWORD", "unused"),
		dockermanage.WithLabel(dockermanage.ManagedLabelKey, "pgdump"),
	)
	if err != nil {
		return nil, fmt.Errorf("start pg_dump container: %w", err)
	}
	defer func() {
		if removeErr := manager.Remove(context.WithoutCancel(ctx), container.ID); removeErr != nil {
			log.Printf("warning: failed to remove container %s: %v", container.ID, removeErr)
		}
	}()

	// Wait briefly for the container to be running (entrypoint starts, pg_dump is available).
	if err := manager.WaitReady(ctx, container, dockerpostgres.TCPReady); err != nil {
		return nil, fmt.Errorf("wait for container: %w", err)
	}

	execCmd := []string{
		"pg_dump",
		"--schema-only",
		"--no-owner",
		"--no-privileges",
	}
	for _, t := range pgdump.DefaultExcludeTables {
		execCmd = append(execCmd, "--exclude-table="+t)
	}
	execCmd = append(execCmd, dbstring)

	var stdout, stderr bytes.Buffer
	result, err := manager.Exec(ctx, container.ID, dockermanage.ExecOptions{
		Cmd: execCmd,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("docker exec pg_dump: %w\n%s", err, stderr.String())
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("pg_dump exited with code %d\n%s", result.ExitCode, stderr.String())
	}
	return stdout.Bytes(), nil
}

// dumpFromMigrations starts an ephemeral Postgres container, applies migrations, dumps the schema,
// and tears everything down.
func dumpFromMigrations(ctx context.Context, migrationsDir string, pgVersion string) ([]byte, error) {
	// Verify migrations directory exists.
	info, err := os.Stat(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("migrations directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", migrationsDir)
	}

	image := fmt.Sprintf("postgres:%s-alpine", pgVersion)

	manager, err := dockermanage.NewManager(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))
	if err != nil {
		return nil, fmt.Errorf("create Docker manager: %w", err)
	}
	defer manager.Close()

	// 1. Start ephemeral Postgres.
	instance, err := dockerpostgres.Start(ctx, manager, dockerpostgres.WithImage(image))
	if err != nil {
		return nil, fmt.Errorf("start ephemeral postgres: %w", err)
	}
	defer func() {
		if removeErr := manager.Remove(context.WithoutCancel(ctx), instance.Container.ID); removeErr != nil {
			log.Printf("warning: failed to remove container %s: %v", instance.Container.ID, removeErr)
		}
	}()

	// 2. Wait for Postgres to accept real connections (not just TCP).
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		instance.Container.Host,
		instance.Container.Port,
		instance.User,
		instance.Password,
		instance.Database,
	)
	if err := manager.WaitReady(ctx, instance.Container, func(ctx context.Context, c *dockermanage.Container) error {
		db, openErr := sql.Open("pgx", dsn)
		if openErr != nil {
			return openErr
		}
		defer db.Close()
		return db.PingContext(ctx)
	}); err != nil {
		return nil, fmt.Errorf("wait for postgres readiness: %w", err)
	}

	// 3. Apply migrations.
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(migrationsDir))
	if err != nil {
		return nil, fmt.Errorf("create migration provider: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	log.Printf("applied %d migration(s)", len(results))

	// 4. Run pg_dump inside the container.
	dumpArgs := pgdump.Args(instance.Database, instance.User)
	var stdout, stderr bytes.Buffer
	result, err := manager.Exec(ctx, instance.Container.ID, dockermanage.ExecOptions{
		Cmd:    dumpArgs,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("docker exec pg_dump: %w\n%s", err, stderr.String())
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("pg_dump exited with code %d\n%s", result.ExitCode, stderr.String())
	}
	return stdout.Bytes(), nil
}

// dumpUsage prints usage information for the dump command.
func dumpUsage() {
	fmt.Fprint(os.Stderr, `Usage: goose dump [OPTIONS]

Export the current PostgreSQL database schema using pg_dump.

Modes:
  Local pg_dump:      goose dump --dbstring "$DSN"
  Docker pg_dump:     goose dump --docker --dbstring "$DSN"
  Ephemeral (CI):     goose dump --docker --from-migrations --dir ./migrations

Options:
`)
	f := flag.NewFlagSet("goose dump", flag.ContinueOnError)
	f.String("dbstring", "", "PostgreSQL connection string (GOOSE_DBSTRING env var supported)")
	f.Bool("docker", false, "use Docker to run pg_dump (and optionally Postgres)")
	f.Bool("from-migrations", false, "spin up ephemeral Postgres, apply migrations, then dump (requires --docker)")
	f.String("dir", ".", "migration files directory (GOOSE_MIGRATION_DIR env var supported)")
	f.Bool("strip", false, "remove comments, SET statements, OWNER TO, and pg_catalog noise")
	f.Bool("goose", false, "add goose annotations (-- +goose Up, StatementBegin/End for $$ blocks)")
	f.String("out", "", "write output to file instead of stdout")
	f.String("pg-version", "16", "PostgreSQL major version for Docker image")
	f.PrintDefaults()
}

