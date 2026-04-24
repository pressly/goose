package dockerpostgres

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pressly/goose/v3/pkg/dockermanage"
)

const (
	// DefaultImage is the default PostgreSQL image.
	DefaultImage = "postgres:16-alpine"

	// DefaultDatabase is the default PostgreSQL database name.
	DefaultDatabase = "testdb"

	// DefaultUser is the default PostgreSQL user.
	DefaultUser = "postgres"

	// DefaultPassword is the default PostgreSQL password.
	DefaultPassword = "password1"

	defaultContainerPort = "5432/tcp"
)

// Option configures a PostgreSQL container instance.
type Option interface {
	apply(*config) error
}

type optionFunc func(*config) error

func (f optionFunc) apply(cfg *config) error {
	return f(cfg)
}

type config struct {
	image    string
	database string
	user     string
	password string
	hostPort int
	labels   map[string]string
}

func defaultConfig() *config {
	return &config{
		image:    DefaultImage,
		database: DefaultDatabase,
		user:     DefaultUser,
		password: DefaultPassword,
		labels: map[string]string{
			dockermanage.ManagedLabelKey: "postgres",
		},
	}
}

// WithImage sets the PostgreSQL image.
func WithImage(image string) Option {
	return optionFunc(func(cfg *config) error {
		image = strings.TrimSpace(image)
		if image == "" {
			return errors.New("image must not be empty")
		}
		cfg.image = image
		return nil
	})
}

// WithDatabase sets the database name.
func WithDatabase(database string) Option {
	return optionFunc(func(cfg *config) error {
		database = strings.TrimSpace(database)
		if database == "" {
			return errors.New("database must not be empty")
		}
		cfg.database = database
		return nil
	})
}

// WithUser sets the database user.
func WithUser(user string) Option {
	return optionFunc(func(cfg *config) error {
		user = strings.TrimSpace(user)
		if user == "" {
			return errors.New("user must not be empty")
		}
		cfg.user = user
		return nil
	})
}

// WithPassword sets the database user password.
func WithPassword(password string) Option {
	return optionFunc(func(cfg *config) error {
		if password == "" {
			return errors.New("password must not be empty")
		}
		cfg.password = password
		return nil
	})
}

// WithHostPort sets a fixed host port. If unset, Docker auto-assigns one.
func WithHostPort(port int) Option {
	return optionFunc(func(cfg *config) error {
		if port <= 0 || port > 65535 {
			return fmt.Errorf("host port must be in range 1-65535: %d", port)
		}
		cfg.hostPort = port
		return nil
	})
}

// WithLabel appends a container label.
func WithLabel(key, value string) Option {
	return optionFunc(func(cfg *config) error {
		key = strings.TrimSpace(key)
		if key == "" {
			return errors.New("label key must not be empty")
		}
		cfg.labels[key] = value
		return nil
	})
}

// WithLabels merges labels into container labels.
func WithLabels(labels map[string]string) Option {
	return optionFunc(func(cfg *config) error {
		for key, value := range maps.Clone(labels) {
			key = strings.TrimSpace(key)
			if key == "" {
				return errors.New("label key must not be empty")
			}
			cfg.labels[key] = value
		}
		return nil
	})
}

// Instance represents a running PostgreSQL container.
type Instance struct {
	Container *dockermanage.Container
	Database  string
	User      string
	Password  string
}

// DSN returns a connection string suitable for PostgreSQL drivers.
func (i *Instance) DSN() string {
	if i == nil || i.Container == nil {
		return ""
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		i.Container.Host,
		i.Container.Port,
		i.User,
		i.Password,
		i.Database,
	)
}

// Start starts a PostgreSQL container and waits for its TCP port to be reachable.
func Start(ctx context.Context, manager *dockermanage.Manager, options ...Option) (_ *Instance, retErr error) {
	if manager == nil {
		return nil, errors.New("manager must not be nil")
	}
	cfg := defaultConfig()
	for _, opt := range options {
		if opt == nil {
			continue
		}
		if err := opt.apply(cfg); err != nil {
			return nil, err
		}
	}

	startOptions := []dockermanage.Option{
		dockermanage.WithImage(cfg.image),
		dockermanage.WithContainerPort(defaultContainerPort),
		dockermanage.WithEnvVars([]string{
			"POSTGRES_DB=" + cfg.database,
			"POSTGRES_USER=" + cfg.user,
			"POSTGRES_PASSWORD=" + cfg.password,
		}),
		dockermanage.WithLabels(cfg.labels),
	}
	if cfg.hostPort > 0 {
		startOptions = append(startOptions, dockermanage.WithHostPort(cfg.hostPort))
	}

	container, err := manager.Start(ctx, startOptions...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, manager.Remove(ctx, container.ID))
		}
	}()

	if err := manager.WaitReady(ctx, container, TCPReady); err != nil {
		return nil, fmt.Errorf("wait for postgres readiness: %w", err)
	}
	return &Instance{
		Container: container,
		Database:  cfg.database,
		User:      cfg.user,
		Password:  cfg.password,
	}, nil
}

// TCPReady checks whether the Postgres TCP port is accepting connections.
func TCPReady(ctx context.Context, c *dockermanage.Container) error {
	if c == nil {
		return errors.New("container must not be nil")
	}
	dialer := net.Dialer{Timeout: 500 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(c.Host, strconv.Itoa(c.Port)))
	if err != nil {
		return err
	}
	return conn.Close()
}
