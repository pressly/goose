package goose

import (
	"errors"
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/lib/pq"
	"os"
	"path/filepath"
)

// DBDriver encapsulates the info needed to work with
// a specific database driver
type DBDriver struct {
	Name    string
	OpenStr string
	Import  string
	Dialect SqlDialect
}

type DBConf struct {
	MigrationsDir string
	Env           string
	Driver        DBDriver
}

// extract configuration details from the given file
func NewDBConf(p, env string) (*DBConf, error) {

	cfgFile := filepath.Join(p, "dbconf.yml")

	f, err := yaml.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	drv, err := f.Get(fmt.Sprintf("%s.driver", env))
	if err != nil {
		return nil, err
	}

	open, err := f.Get(fmt.Sprintf("%s.open", env))
	if err != nil {
		return nil, err
	}
	open = os.ExpandEnv(open)

	// Automatically parse postgres urls
	if drv == "postgres" {

		// Assumption: If we can parse the URL, we should
		if parsedURL, err := pq.ParseURL(open); err == nil && parsedURL != "" {
			open = parsedURL
		}
	}

	d := newDBDriver(drv, open)

	// allow the configuration to override the Import for this driver
	if imprt, err := f.Get(fmt.Sprintf("%s.import", env)); err == nil {
		d.Import = imprt
	}

	// allow the configuration to override the Dialect for this driver
	if dialect, err := f.Get(fmt.Sprintf("%s.dialect", env)); err == nil {
		d.Dialect = dialectByName(dialect)
	}

	if !d.IsValid() {
		return nil, errors.New(fmt.Sprintf("Invalid DBConf: %v", d))
	}

	return &DBConf{
		MigrationsDir: filepath.Join(p, "migrations"),
		Env:           env,
		Driver:        d,
	}, nil
}

// Create a new DBDriver and populate driver specific
// fields for drivers that we know about.
// Further customization may be done in NewDBConf
func newDBDriver(name, open string) DBDriver {

	d := DBDriver{
		Name:    name,
		OpenStr: open,
	}

	switch name {
	case "postgres":
		d.Import = "github.com/lib/pq"
		d.Dialect = &PostgresDialect{}

	case "mymysql":
		d.Import = "github.com/ziutek/mymysql/godrv"
		d.Dialect = &MySqlDialect{}
	}

	return d
}

// ensure we have enough info about this driver
func (drv *DBDriver) IsValid() bool {
	return len(drv.Import) > 0 && drv.Dialect != nil
}
