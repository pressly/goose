package testdb

type options struct {
	bindPort int
	debug    bool
	folder   string
}

type OptionsFunc func(o *options)

func WithBindPort(n int) OptionsFunc {
	return func(o *options) { o.bindPort = n }
}

func WithDebug(b bool) OptionsFunc {
	return func(o *options) { o.debug = b }
}

func WithFolder(f string) OptionsFunc {
	return func(o *options) { o.folder = f }
}
