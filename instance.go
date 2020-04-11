package goose

var def = NewInstance()

// Instance of goose for managing single database / migration directory.
// Use if you've more than one database / migration directory compiled in
// single Go binary.
type Instance struct {
	dialect                SQLDialect
	registeredGoMigrations map[int64]*Migration
	tableName              string
	log                    Logger
	verbose                bool
}

// NewInstance creates and returns new goose instance.
func NewInstance() *Instance {
	in := &Instance{}
	in.dialect = &PostgresDialect{tableName: in.TableName}
	in.registeredGoMigrations = make(map[int64]*Migration)
	in.tableName = "goose_db_version"
	in.log = &stdLogger{}
	in.verbose = false
	return in
}
