package goose

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	seqVersionFormat = "%05v"
)

type CreateOptions struct {
	Sequential bool
	NoTx       bool
	Template   *template.Template

	timeFunc func() time.Time
}

// Create creates a new migration file in the specified directory. If the directory does not exist,
// it will be created. This command always reads from and writes to the filesystem. The returned
// filename is the full path to the newly created migration file.
//
// By default, the filename numeric component will be a timestamp in the format of YYYYMMDDHHMMSS.
// But if sequential is set to true, it will be the next available sequential number. Example:
// 00001_create_users_table.go.
//
// The provided CreateOptions is optional and may be nil if defaults should be used.
func Create(
	dir string,
	migrationType MigrationType,
	name string,
	opt *CreateOptions,
) (string, error) {
	if opt == nil {
		opt = new(CreateOptions)
	}
	if dir == "" {
		return "", fmt.Errorf("dir cannot be empty")
	}
	switch migrationType {
	case MigrationTypeGo, MigrationTypeSQL:
	default:
		return "", fmt.Errorf("invalid migration type: %v", migrationType)
	}
	now := time.Now()
	if opt.timeFunc != nil {
		now = opt.timeFunc()
	} else {
		now = time.Now()
	}
	version := now.Format(timestampFormat)
	if opt.Sequential {
		// TODO(mf): do not parse all the files, no need to do this in this case.
		migrations, err := collectMigrations(registeredGoMigrations, osFS{}, dir, false, nil)
		if err != nil {
			return "", err
		}
		vMigrations, err := versioned(migrations)
		if err != nil {
			return "", err
		}
		var v int64 = 1
		if len(vMigrations) > 0 {
			v = vMigrations[len(vMigrations)-1].version + 1
		}
		version = fmt.Sprintf(seqVersionFormat, v)
	}
	filename := fmt.Sprintf("%v_%v.%v", version, snakeCase(name), string(migrationType))

	tmpl := opt.Template
	if tmpl == nil {
		var dat string
		switch migrationType {
		case MigrationTypeGo:
			var goFunc string
			var param string
			if opt.NoTx {
				goFunc = "AddMigrationNoTx"
				param = "db *sql.DB"
			} else {
				goFunc = "AddMigration"
				param = "tx *sql.Tx"
			}
			dat = fmt.Sprintf(goMigrationStr, goFunc, param, param)
		case MigrationTypeSQL:
			dat = sqlMigrationStr
			if opt.NoTx {
				dat = "-- +goose NO TRANSACTION\n\n" + dat
			}
		}
		var err error
		tmpl, err = template.New("").Parse(dat)
		if err != nil {
			return "", err
		}
	}

	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to create migration file: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("failed to create migration file: %w", err)
	}
	defer f.Close()

	vars := struct {
		Version   string
		CamelName string
	}{
		Version:   version,
		CamelName: camelCase(name),
	}
	if err := tmpl.Execute(f, vars); err != nil {
		return "", fmt.Errorf("failed to execute tmpl: %w", err)
	}
	return f.Name(), nil
}

const (
	sqlMigrationStr = `-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd`
)

const (
	goMigrationStr = `package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	// {{.CamelName}}
	goose.%s(up{{.Version}}, down{{.Version}})
}

func up{{.Version}}(%s) error {
	// This code is executed when the migration is applied.
	return nil
}

func down{{.Version}}(%s) error {
	// This code is executed when the migration is rolled back.
	return nil
}`
)

func versioned(in []*migration) ([]*migration, error) {
	var migrations []*migration
	// assume that the user will never have more than 19700101000000 migrations
	for _, m := range in {
		// parse version as timestamp
		versionTime, err := time.Parse(timestampFormat, strconv.FormatInt(m.version, 10))
		if versionTime.Before(time.Unix(0, 0)) || err != nil {
			migrations = append(migrations, m)
		}
	}
	return migrations, nil
}

type camelSnakeStateMachine int

const ( //                                           _$$_This is some text, OK?!
	idle          camelSnakeStateMachine = iota // 0 ↑                     ↑   ↑
	firstAlphaNum                               // 1     ↑    ↑  ↑    ↑     ↑
	alphaNum                                    // 2      ↑↑↑  ↑  ↑↑↑  ↑↑↑   ↑
	delimiter                                   // 3         ↑  ↑    ↑    ↑   ↑
)

func (s camelSnakeStateMachine) next(r rune) camelSnakeStateMachine {
	switch s {
	case idle:
		if isAlphaNum(r) {
			return firstAlphaNum
		}
	case firstAlphaNum:
		if isAlphaNum(r) {
			return alphaNum
		}
		return delimiter
	case alphaNum:
		if !isAlphaNum(r) {
			return delimiter
		}
	case delimiter:
		if isAlphaNum(r) {
			return firstAlphaNum
		}
		return idle
	}
	return s
}

func camelCase(str string) string {
	var b strings.Builder

	stateMachine := idle
	for i := 0; i < len(str); {
		r, size := utf8.DecodeRuneInString(str[i:])
		i += size
		stateMachine = stateMachine.next(r)
		switch stateMachine {
		case firstAlphaNum:
			b.WriteRune(unicode.ToUpper(r))
		case alphaNum:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func snakeCase(str string) string {
	var b bytes.Buffer

	stateMachine := idle
	for i := 0; i < len(str); {
		r, size := utf8.DecodeRuneInString(str[i:])
		i += size
		stateMachine = stateMachine.next(r)
		switch stateMachine {
		case firstAlphaNum, alphaNum:
			b.WriteRune(unicode.ToLower(r))
		case delimiter:
			b.WriteByte('_')
		}
	}
	if stateMachine == idle {
		return string(bytes.TrimSuffix(b.Bytes(), []byte{'_'}))
	}
	return b.String()
}

func isAlphaNum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}
