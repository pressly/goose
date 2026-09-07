package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/multierr"
)

// NewSqlserver creates a new SQL Server-based [LockStore].
func NewSqlserver(tableName string) (LockStore, error) {
	if tableName == "" {
		return nil, errors.New("table name must not be empty")
	}
	return &sqlserverStore{
		tableName: tableName,
	}, nil
}

var _ LockStore = (*sqlserverStore)(nil)

type sqlserverStore struct {
	tableName string
}

func (s *sqlserverStore) TableExists(
	ctx context.Context,
	db *sql.DB,
) (bool, error) {
	var query string
	schemaName, tableName := parseSqlserverTableIdentifier(s.tableName)
	if schemaName != "" {
		q := `SELECT CASE WHEN EXISTS (
			SELECT 1 FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'
		) THEN 1 ELSE 0 END`
		query = fmt.Sprintf(q, schemaName, tableName)
	} else {
		q := `SELECT CASE WHEN EXISTS (
			SELECT 1 FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = SCHEMA_NAME() AND TABLE_NAME = '%s'
		) THEN 1 ELSE 0 END`
		query = fmt.Sprintf(q, tableName)
	}

	var exists int
	if err := db.QueryRowContext(ctx, query).Scan(&exists); err != nil {
		return false, fmt.Errorf("table exists: %w", err)
	}
	return exists == 1, nil
}

func (s *sqlserverStore) CreateLockTable(
	ctx context.Context,
	db *sql.DB,
) error {
	exists, err := s.TableExists(ctx, db)
	if err != nil {
		return fmt.Errorf("check lock table existence: %w", err)
	}
	if exists {
		return nil
	}

	query := fmt.Sprintf(`CREATE TABLE %s (
		lock_id BIGINT NOT NULL PRIMARY KEY,
		locked BIT NOT NULL DEFAULT 0,
		locked_at DATETIMEOFFSET NULL,
		locked_by NVARCHAR(255) NULL,
		lease_expires_at DATETIMEOFFSET NULL,
		updated_at DATETIMEOFFSET NULL
	)`, s.tableName)
	if _, err := db.ExecContext(ctx, query); err != nil {
		// Double-check if another process created it concurrently
		if exists, checkErr := s.TableExists(ctx, db); checkErr == nil && exists {
			// Another process created it, that's fine!
			return nil
		}
		return fmt.Errorf("create lock table %q: %w", s.tableName, err)
	}
	return nil
}

func (s *sqlserverStore) AcquireLock(
	ctx context.Context,
	db *sql.DB,
	lockID int64,
	lockedBy string,
	leaseDuration time.Duration,
) (*AcquireLockResult, error) {
	// SQL Server uses MERGE for upsert functionality
	// We only update if the lock is not held (locked = 0) or the lease has expired
	query := fmt.Sprintf(`
		MERGE %s WITH (HOLDLOCK) AS target
		USING (SELECT @p1 AS lock_id) AS source
		ON target.lock_id = source.lock_id
		WHEN MATCHED AND (target.locked = 0 OR target.lease_expires_at < SYSUTCDATETIME()) THEN
			UPDATE SET
				locked = 1,
				locked_at = SYSUTCDATETIME(),
				locked_by = @p2,
				lease_expires_at = DATEADD(SECOND, @p3, SYSUTCDATETIME()),
				updated_at = SYSUTCDATETIME()
		WHEN NOT MATCHED THEN
			INSERT (lock_id, locked, locked_at, locked_by, lease_expires_at, updated_at)
			VALUES (@p1, 1, SYSUTCDATETIME(), @p2, DATEADD(SECOND, @p3, SYSUTCDATETIME()), SYSUTCDATETIME())
		OUTPUT inserted.locked_by, inserted.lease_expires_at;`,
		s.tableName)

	// Convert duration to seconds
	leaseDurationSeconds := int(leaseDuration.Seconds())

	var returnedLockedBy string
	var leaseExpiresAt time.Time
	err := db.QueryRowContext(ctx, query,
		lockID,
		lockedBy,
		leaseDurationSeconds,
	).Scan(
		&returnedLockedBy,
		&leaseExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("acquire lock %d: already held by another instance", lockID)
		}
		return nil, fmt.Errorf("acquire lock %d: %w", lockID, err)
	}

	// Verify we got the lock by checking the returned locked_by matches our instance ID
	if returnedLockedBy != lockedBy {
		return nil, fmt.Errorf("acquire lock %d: acquired by %s instead of %s", lockID, returnedLockedBy, lockedBy)
	}

	return &AcquireLockResult{
		LockedBy:       returnedLockedBy,
		LeaseExpiresAt: leaseExpiresAt,
	}, nil
}

func (s *sqlserverStore) ReleaseLock(
	ctx context.Context,
	db *sql.DB,
	lockID int64,
	lockedBy string,
) (*ReleaseLockResult, error) {
	// Release lock only if it's held by the current instance
	query := fmt.Sprintf(`UPDATE %s SET
		locked = 0,
		locked_at = NULL,
		locked_by = NULL,
		lease_expires_at = NULL,
		updated_at = SYSUTCDATETIME()
	OUTPUT inserted.lock_id
	WHERE lock_id = @p1 AND locked_by = @p2`, s.tableName)

	var returnedLockID int64
	err := db.QueryRowContext(ctx, query,
		lockID,
		lockedBy,
	).Scan(
		&returnedLockID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("release lock %d: not held by this instance", lockID)
		}
		return nil, fmt.Errorf("release lock %d: %w", lockID, err)
	}

	// Verify the correct lock was released
	if returnedLockID != lockID {
		return nil, fmt.Errorf("release lock %d: returned lock ID %d does not match", lockID, returnedLockID)
	}

	return &ReleaseLockResult{
		LockID: returnedLockID,
	}, nil
}

func (s *sqlserverStore) UpdateLease(
	ctx context.Context,
	db *sql.DB,
	lockID int64,
	lockedBy string,
	leaseDuration time.Duration,
) (*UpdateLeaseResult, error) {
	// Update lease expiration time for heartbeat, only if we own the lock
	query := fmt.Sprintf(`UPDATE %s SET
		lease_expires_at = DATEADD(SECOND, @p1, SYSUTCDATETIME()),
		updated_at = SYSUTCDATETIME()
	OUTPUT inserted.lease_expires_at
	WHERE lock_id = @p2 AND locked_by = @p3 AND locked = 1`, s.tableName)

	// Convert duration to seconds
	leaseDurationSeconds := int(leaseDuration.Seconds())

	var leaseExpiresAt time.Time
	err := db.QueryRowContext(ctx, query,
		leaseDurationSeconds,
		lockID,
		lockedBy,
	).Scan(
		&leaseExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to update lease for lock %d: not held by this instance", lockID)
		}
		return nil, fmt.Errorf("failed to update lease for lock %d: %w", lockID, err)
	}

	return &UpdateLeaseResult{
		LeaseExpiresAt: leaseExpiresAt,
	}, nil
}

func (s *sqlserverStore) CheckLockStatus(
	ctx context.Context,
	db *sql.DB,
	lockID int64,
) (*LockStatus, error) {
	query := fmt.Sprintf(`SELECT locked, locked_by, lease_expires_at, updated_at FROM %s WHERE lock_id = @p1`, s.tableName)
	var status LockStatus

	err := db.QueryRowContext(ctx, query,
		lockID,
	).Scan(
		&status.Locked,
		&status.LockedBy,
		&status.LeaseExpiresAt,
		&status.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("lock %d not found", lockID)
		}
		return nil, fmt.Errorf("check lock status for %d: %w", lockID, err)
	}

	return &status, nil
}

func (s *sqlserverStore) CleanupStaleLocks(ctx context.Context, db *sql.DB) (_ []int64, retErr error) {
	query := fmt.Sprintf(`UPDATE %s SET
		locked = 0,
		locked_at = NULL,
		locked_by = NULL,
		lease_expires_at = NULL,
		updated_at = SYSUTCDATETIME()
	OUTPUT inserted.lock_id
	WHERE locked = 1 AND lease_expires_at < SYSUTCDATETIME()`, s.tableName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("cleanup stale locks: %w", err)
	}
	defer func() {
		retErr = multierr.Append(retErr, rows.Close())
	}()

	var cleanedLocks []int64
	for rows.Next() {
		var lockID int64
		if err := rows.Scan(&lockID); err != nil {
			return nil, fmt.Errorf("scan cleaned lock ID: %w", err)
		}
		cleanedLocks = append(cleanedLocks, lockID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate over cleaned locks: %w", err)
	}

	return cleanedLocks, nil
}

func parseSqlserverTableIdentifier(name string) (schema, table string) {
	schema, table, found := strings.Cut(name, ".")
	if !found {
		return "", name
	}
	return schema, table
}
