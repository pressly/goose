package pgutil

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func Dump(options ConnectionOptions, clean bool) (string, error) {
	var b bytes.Buffer
	if err := dump(&b, options); err != nil {
		return "", err
	}
	if clean {
		return cleanup(&b)
	}
	return b.String(), nil
}

var matchIgnorePrefix = []string{
	"--",
	"COMMENT ON",
	"REVOKE",
	"GRANT",
	"SET",
	"ALTER DEFAULT PRIVILEGES",
}

var matchIgnoreContains = []string{
	"ALTER DEFAULT PRIVILEGES",
	"OWNER TO",
}

func ignore(input string) bool {
	for _, v := range matchIgnorePrefix {
		if strings.HasPrefix(input, v) {
			return true
		}
	}
	for _, v := range matchIgnoreContains {
		if strings.Contains(input, v) {
			return true
		}
	}
	return false
}

var reAdjacentEmptyLines = regexp.MustCompile(`(?m)(\n){3,}`)

func cleanup(r io.Reader) (string, error) {
	var b strings.Builder
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if ignore(line) {
			continue
		}
		if _, err := b.WriteString(line + "\n"); err != nil {
			return "", fmt.Errorf("failed to write output: %w", err)
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	result := reAdjacentEmptyLines.ReplaceAllString(b.String(), "\n\n")
	return "\n" + strings.TrimSpace(result) + "\n", nil
}

func dump(w io.Writer, options ConnectionOptions) error {
	pgDumpPath, err := exec.LookPath("pg_dump")
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return err
	}
	if pgDumpPath != "" {
		return runPgDump(w, pgDumpPath, options)
	}
	dockerPath, err := exec.LookPath("docker")
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return err
	}
	if dockerPath != "" {
		return runDockerPgDump(dockerPath)
	}
	return fmt.Errorf("failed to find at least one exectuable: pg_dump, docker")
}

type ConnectionOptions struct {
	DBname   string
	Username string
	Host     string
	Port     string
	Password string
}

func NewConnectionOptions(raw string) (ConnectionOptions, error) {
	var opt ConnectionOptions
	u, err := url.Parse(raw)
	if err != nil {
		return opt, err
	}
	if u.User == nil {
		return opt, fmt.Errorf("invalid postgres connection string: missing username and password information")
	}
	// TODO(mf): we could ask the user for password with a prompt here.
	pass, ok := u.User.Password()
	if !ok {
		return opt, fmt.Errorf("invalid postgres connection string: missing password information")
	}
	opt.Username = u.User.Username()
	opt.Password = pass
	opt.Host = u.Hostname()
	opt.Port = u.Port()
	opt.DBname = strings.TrimPrefix(u.Path, "/")
	// TODO(mf): What about u.RawQuery to get at options after the ? .."sslmode=disable"
	return opt, nil
}

func (c ConnectionOptions) Args() []string {
	return []string{
		"--dbname=" + c.DBname,
		"--host=" + c.Host,
		"--port=" + c.Port,
		"--username=" + c.Username,
	}
}

func runPgDump(w io.Writer, pgDumpPath string, connOptions ConnectionOptions) error {
	args := append(connOptions.Args(), "--schema-only")
	cmd := exec.Command(pgDumpPath, args...)
	cmd.Env = append(cmd.Env, "PGPASSWORD="+connOptions.Password)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	return cmd.Run()

}

func runDockerPgDump(dockerPath string) error {
	return errors.New("unimplemented")
}
