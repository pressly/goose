package goose

import "database/sql"

// StandAlone contains wrapper methods to that can be configured and called from within go
type StandAlone struct {
	cfg *config
}

// New creates a pointer StandAlone structure
func New(dir string, db *sql.DB, opts ...Option) *StandAlone {
	c := newConfig(dir, db, opts...)
	return &StandAlone{cfg: c}
}

// Down rolls back a single migration from the current version 
func (s StandAlone) Down() error {
	return Down(s.cfg)
}

// DownTo rolls back migrations to a specific version
func (s StandAlone) DownTo(version int64) error {
	return DownTo(s.cfg, version)
}

// Fix ...
func (s StandAlone) Fix() error {
	return Fix(s.cfg)
}

// Redo rolls back the most recently applied migration, then runs it again
func (s StandAlone) Redo() error {
	return Redo(s.cfg)
}

// Reset rolls back all migrations
func (s StandAlone) Reset() error {
	return Reset(s.cfg)
}

// Status prints the status of all migrations
func (s StandAlone) Status() error {
	return Status(s.cfg)
}

// Up applies all available migrations
func (s StandAlone) Up() error {
	return Up(s.cfg)
}

// UpByOne migrates up by a single version
func (s StandAlone) UpByOne() error {
	return UpByOne(s.cfg)
}

// UpTo migrates up to a specific version
func (s StandAlone) UpTo(version int64) error {
	return UpTo(s.cfg, version)
}

// Version prints the current version of the database
func (s StandAlone) Version() error {
	return Version(s.cfg)
}