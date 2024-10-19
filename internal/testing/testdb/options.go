package testdb

import "github.com/ory/dockertest/v3"

type options struct {
	env      []string
	mounts   []string
	network  *dockertest.Network
	name     string
	bindPort int
	debug    bool
}

type OptionsFunc func(o *options)

func WithBindPort(n int) OptionsFunc {
	return func(o *options) { o.bindPort = n }
}

func WithDebug(b bool) OptionsFunc {
	return func(o *options) { o.debug = b }
}

func WithMounts(m []string) OptionsFunc {
	return func(o *options) { o.mounts = m }
}

func WithName(n string) OptionsFunc {
	return func(o *options) { o.name = n }
}

func WithEnv(e []string) OptionsFunc {
	return func(o *options) { o.env = e }
}

func WithNetwork(n *dockertest.Network) OptionsFunc {
	return func(o *options) { o.network = n }
}
