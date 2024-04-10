package cli

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
)

// state holds the state of the CLI and is passed to each command. It is used to configure the
// environment, filesystem, and output streams.
type state struct {
	version string
	environ []string
	stdout  io.Writer
	stderr  io.Writer
	// This is effectively [fs.SubFS](https://pkg.go.dev/io/fs#SubFS).
	fsys           func(dir string) (fs.FS, error)
	openConnection func(dbstring string) (*sql.DB, goose.Dialect, error)
}

func (s *state) writeJSON(v interface{}) error {
	by, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = s.stdout.Write(by)
	return err
}

func (s *state) initProvider(
	dir string,
	dbstring string,
	tablename string,
	options ...goose.ProviderOption,
) (*goose.Provider, error) {
	if dir == "" {
		return nil, fmt.Errorf("migrations directory is required, set with --dir or GOOSE_DIR")
	}
	if dbstring == "" {
		return nil, errors.New("database connection string is required, set with --dbstring or GOOSE_DBSTRING")
	}
	db, dialect, err := openConnection(dbstring)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	if tablename != "" {
		store, err := database.NewStore(dialect, tablename)
		if err != nil {
			return nil, fmt.Errorf("failed to create store: %w", err)
		}
		options = append(options, goose.WithStore(store))
		// TODO(mf): I don't like how this works. It's not obvious that if a store is provided, the
		// dialect must be set to an empty string. This is because the dialect is set in the store.
		dialect = ""
	}
	fsys, err := s.fsys(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtree rooted at dir: %q: %w", dir, err)
	}
	return goose.NewProvider(dialect, db, fsys, options...)
}
