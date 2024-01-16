package gomigrations

import (
	"github.com/piiano/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(nil, nil)
}
