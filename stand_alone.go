package goose

// StandAlone contains wrapper methods to that can be configured and called from within go
type StandAlone struct {
	opts *options
}

// New creates a pointer StandAlone structure
func New(opts *options) *StandAlone {
	return &StandAlone{opts: opts}
}

// Down rolls back a single migration from the current version 
func (s StandAlone) Down() error {
	return Down(s.opts)
}

// DownTo rolls back migrations to a specific version
func (s StandAlone) DownTo(version int64) error {
	return DownTo(s.opts, version)
}

// Fix ...
func (s StandAlone) Fix() error {
	return Fix(s.opts)
}

// Redo rolls back the most recently applied migration, then runs it again
func (s StandAlone) Redo() error {
	return Redo(s.opts)
}

// Reset rolls back all migrations
func (s StandAlone) Reset() error {
	return Reset(s.opts)
}

// Status prints the status of all migrations
func (s StandAlone) Status() error {
	return Status(s.opts)
}

// Up applies all available migrations
func (s StandAlone) Up() error {
	return Up(s.opts)
}

// UpByOne migrates up by a single version
func (s StandAlone) UpByOne() error {
	return UpByOne(s.opts)
}

// UpTo migrates up to a specific version
func (s StandAlone) UpTo(version int64) error {
	return UpTo(s.opts, version)
}

// Version prints the current version of the database
func (s StandAlone) Version() error {
	return Version(s.opts)
}