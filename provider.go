package goose

import (
	"io/fs"
	"path/filepath"
	"runtime"
	"time"
)

const (
	defaultProviderPackage = "migrations"
	defaultTableName       = "goose_db_version"
	defaultTimestampFormat = "20060102150405"
)

// defaultProvider is the provider the general functions use.
var defaultProvider = NewProvider()

type providerOptions func(p *Provider)

// TimestampFormat sets the timestamp format for the provider
func TimestampFormat(format string) func(p *Provider) {
	return func(p *Provider) {
		p.timestampFormat = format
	}
}

// TimeFunction sets the time function used to get the time for timestamp numbers
// defaults to time.Now
func TimeFunction(fn func() time.Time) func(p *Provider) {
	if fn == nil {
		fn = time.Now
	}
	return func(p *Provider) {
		p.timeFn = fn
	}
}

// Verbose sets the verbose on the provider
func Verbose(b bool) func(p *Provider) {
	return func(p *Provider) {
		p.verbose = b
	}
}

// SequentialVersion make the provider use sequential versioning
func SequentialVersion(versionTemplate string) func(p *Provider) {
	return func(p *Provider) {
		p.sequential = true
		if versionTemplate != "" {
			p.seqVersionTemplate = versionTemplate
		}
	}
}

// TimestampVersion make the provider use sequential versioning
func TimestampVersion(p *Provider) {
	p.sequential = false
}
func Filesystem(baseFS fs.FS) func(p *Provider) {
	return func(p *Provider) {
		p.baseFS = baseFS
	}
}

func Log(log Logger) func(p *Provider) {
	return func(p *Provider) {
		p.log = log
	}
}

func Dialect(dialect string) func(p *Provider) {
	return func(p *Provider) {
		dialect, err := SelectDialect(p.tableName, dialect)
		if err != nil {
			p.log.Fatal(err)
		}
		p.dialect = dialect
	}
}

// dirPath finds the directory path of the calling function's caller
func dirPath() string {
	_, filename, _, _ := runtime.Caller(2)
	return filepath.Dir(filename)
}

// BaseDir will set the base directory, if an empty string is passed
// the directory of the package that called BaseDir is used instead
// this is only useful for Create* and Fix functions
func BaseDir(dir string) func(p *Provider) {
	if dir == "" {
		dir = dirPath()
	}
	return func(p *Provider) {
		p.baseDir = dir
	}
}

func DialectObject(dialect SQLDialect) func(p *Provider) {
	return func(p *Provider) {
		p.dialect = dialect
		p.dialect.SetTableName(p.tableName)
	}
}

func Tablename(tablename string) func(p *Provider) {
	return func(p *Provider) {
		p.tableName = tablename
		p.dialect.SetTableName(tablename)
	}
}

// ProviderPackage sets the packageName and providerVar used in templates
func ProviderPackage(packageName, providerVar string) func(p *Provider) {
	if packageName == "" {
		packageName = defaultProviderPackage
	}
	return func(p *Provider) {
		p.packageName = packageName
		p.providerVarName = providerVar
	}
}

type Provider struct {
	timestampFormat string
	// defaults to time.Now
	timeFn  func() time.Time
	verbose bool
	// whether to use sequential versioning instead of timestamp based versioning
	sequential             bool
	baseFS                 fs.FS
	log                    Logger
	dialect                SQLDialect
	registeredGoMigrations map[int64]*Migration
	tableName              string
	// seqVersionTemplate sets the template system will use this to format the digit of the sequence number
	// by default it %05d, see seqVersionTemplate for actually default value.
	seqVersionTemplate string
	// packageName is the name of the package to use for Create functions
	packageName string
	// providerVarName is the name of the provider var for create functions
	providerVarName string
	// This is used for Create/Fix if the dir is not passed.
	baseDir string
}

func NewProvider(options ...providerOptions) *Provider {
	p := &Provider{
		timestampFormat:        defaultTimestampFormat,
		timeFn:                 time.Now,
		verbose:                false,
		sequential:             false,
		baseFS:                 osFS{},
		log:                    log,
		dialect:                &PostgresDialect{},
		registeredGoMigrations: map[int64]*Migration{},
		tableName:              defaultTableName,
		packageName:            defaultProviderPackage,
	}
	for _, opt := range options {
		opt(p)
	}
	return p
}
