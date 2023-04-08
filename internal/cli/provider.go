package cli

import (
	"fmt"

	"github.com/pressly/goose/v4"
)

func newGooseProvider(root *rootConfig) (*goose.Provider, error) {
	db, gooseDialect, err := openConnection(root.dbstring)
	if err != nil {
		return nil, err
	}
	opt := goose.DefaultOptions().
		SetVerbose(root.verbose).
		SetNoVersioning(root.noVersioning).
		SetAllowMissing(root.allowMissing)

	if len(root.excludeFilenames) > 0 {
		opt = opt.SetExcludeFilenames(root.excludeFilenames...)
	}
	if root.dir != "" {
		opt = opt.SetDir(root.dir)
	}
	if root.table != "" {
		opt = opt.SetTableName(root.table)
	}
	if root.lockMode != "" {
		var lockMode goose.LockMode
		switch root.lockMode {
		case "none":
			lockMode = goose.LockModeNone
		case "advisory-session":
			lockMode = goose.LockModeAdvisorySession
		default:
			return nil, fmt.Errorf("invalid lock mode: %s", root.lockMode)
		}
		opt = opt.SetLockMode(lockMode)
	}

	return goose.NewProvider(gooseDialect, db, opt)
}
