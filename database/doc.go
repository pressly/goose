// Package database provides a Store interface for goose to use when interacting with the database.
// It also provides a an implementation for each supported database dialect.
//
// The Store interface is meant to be generic and not tied to any specific database.
//
// It's possible to implement a custom Store for a database that goose does not support. To do so,
// implement the [Store] interface and pass it to [goose.NewProvider].
package database
