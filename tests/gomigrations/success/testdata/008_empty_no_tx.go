package gomigrations

import (
	"github.com/piiano/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(nil, nil)
}
