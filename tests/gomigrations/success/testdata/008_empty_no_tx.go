package gomigrations

import (
	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigrationNoTx(nil, nil)
}
