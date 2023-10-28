package goose

// MigrationType is the type of migration.
type MigrationType string

const (
	TypeGo  MigrationType = "go"
	TypeSQL MigrationType = "sql"
)

func (t MigrationType) String() string {
	// This should never happen.
	if t == "" {
		return "unknown migration type"
	}
	return string(t)
}
