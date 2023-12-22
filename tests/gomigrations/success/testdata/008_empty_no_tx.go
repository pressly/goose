package gomigrations

import (
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(nil, nil)
}
