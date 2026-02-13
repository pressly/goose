package dockerpostgres_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3/pkg/dockermanage"
	"github.com/pressly/goose/v3/pkg/dockermanage/dockerpostgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newManager(t *testing.T) *dockermanage.Manager {
	t.Helper()
	m, err := dockermanage.NewManager(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, m.Close())
	})
	return m
}

func TestStartAndConnect(t *testing.T) {
	t.Parallel()

	m := newManager(t)
	ctx := t.Context()

	instance, err := dockerpostgres.Start(ctx, m)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, m.Remove(context.WithoutCancel(ctx), instance.Container.ID))
	})

	require.Positive(t, instance.Container.Port)
	require.NotEmpty(t, instance.Container.Host)

	// TCP readiness doesn't guarantee Postgres is accepting queries yet. Wait for a real ping.
	err = m.WaitReady(ctx, instance.Container, func(ctx context.Context, c *dockermanage.Container) error {
		conn, err := pgx.Connect(ctx, instance.DSN())
		if err != nil {
			return err
		}
		return errors.Join(conn.Ping(ctx), conn.Close(ctx))
	})
	require.NoError(t, err)
}

func TestStartAndStop(t *testing.T) {
	t.Parallel()

	m := newManager(t)
	ctx := t.Context()

	instance, err := dockerpostgres.Start(ctx, m)
	require.NoError(t, err)

	require.NoError(t, m.Stop(ctx, instance.Container.ID))
	require.NoError(t, m.Remove(ctx, instance.Container.ID))
}

func TestManagedLabel(t *testing.T) {
	t.Parallel()

	m := newManager(t)
	ctx := t.Context()

	instance, err := dockerpostgres.Start(ctx, m)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, m.Remove(context.WithoutCancel(ctx), instance.Container.ID))
	})

	label, ok := instance.Container.Labels[dockermanage.ManagedLabelKey]
	require.True(t, ok, "expected managed label to be set")
	require.Equal(t, "postgres", label)

	ids, err := m.ListManaged(ctx)
	require.NoError(t, err)
	require.Contains(t, ids, instance.Container.ID)
}

func TestExecPgDump(t *testing.T) {
	t.Parallel()

	m := newManager(t)
	ctx := t.Context()

	instance, err := dockerpostgres.Start(ctx, m)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, m.Remove(context.WithoutCancel(ctx), instance.Container.ID))
	})

	// Wait until Postgres is actually accepting connections.
	err = m.WaitReady(ctx, instance.Container, func(ctx context.Context, c *dockermanage.Container) error {
		conn, err := pgx.Connect(ctx, instance.DSN())
		if err != nil {
			return err
		}
		return errors.Join(conn.Ping(ctx), conn.Close(ctx))
	})
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	result, err := m.Exec(ctx, instance.Container.ID, dockermanage.ExecOptions{
		Cmd: []string{
			"pg_dump",
			"-U", instance.User,
			"-d", instance.Database,
			"--schema-only",
		},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	require.NoError(t, err)
	require.Equal(t, 0, result.ExitCode, "stderr: %s", stderr.String())
	require.Contains(t, stdout.String(), "PostgreSQL database dump")
	require.Contains(t, stdout.String(), "PostgreSQL database dump complete")
}
