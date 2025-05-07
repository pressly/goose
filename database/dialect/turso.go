package dialect

// NewTurso returns a [Querier] for Turso dialect.
func NewTurso() Querier {
	return &turso{}
}

type turso struct {
	sqlite3
}

var _ Querier = (*turso)(nil)
