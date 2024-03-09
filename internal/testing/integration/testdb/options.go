package testdb

type options struct {
	bindPort int
	debug    bool
	// for embedded databases
	databaseFile string
}

type OptionsFunc func(o *options)

func WithBindPort(n int) OptionsFunc {
	return func(o *options) { o.bindPort = n }
}

func WithDebug(b bool) OptionsFunc {
	return func(o *options) { o.debug = b }
}

func WithDatabaseFile(p string) OptionsFunc {
	return func(o *options) { o.databaseFile = p }
}
