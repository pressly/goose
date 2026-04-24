package dockermanage

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/moby/moby/api/types/network"
)

const (
	// DefaultHostIP is the default host IP used for port bindings.
	DefaultHostIP = "127.0.0.1"

	// ManagedLabelKey marks containers created by this package. The value indicates the container
	// type (e.g., "postgres"). Presence of the key means the container is managed.
	ManagedLabelKey = "pressly.goose"
)

// Option configures container start behavior.
type Option interface {
	apply(*config) error
}

type optionFunc func(*config) error

func (f optionFunc) apply(cfg *config) error {
	return f(cfg)
}

type config struct {
	name          string
	image         string
	containerPort network.Port
	hostIP        string
	hostPort      int
	envVars       []string
	autoRemove    bool
	pullProgress  io.Writer
	labels        map[string]string
}

func defaultConfig() *config {
	return &config{
		hostIP:  DefaultHostIP,
		envVars: []string{},
		labels: map[string]string{
			ManagedLabelKey: "",
		},
	}
}

// WithName sets the container name.
func WithName(name string) Option {
	return optionFunc(func(cfg *config) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return errors.New("container name must not be empty")
		}
		cfg.name = name
		return nil
	})
}

// WithImage sets the container image (for example: postgres:16-alpine).
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

// WithContainerPort sets the container port to expose, for example: "5432/tcp".
func WithContainerPort(port string) Option {
	return optionFunc(func(cfg *config) error {
		p, err := network.ParsePort(port)
		if err != nil {
			return fmt.Errorf("invalid container port: %w", err)
		}
		cfg.containerPort = p
		return nil
	})
}

// WithContainerPortTCP is a convenience helper for TCP ports.
func WithContainerPortTCP(port int) Option {
	return optionFunc(func(cfg *config) error {
		if port <= 0 || port > 65535 {
			return fmt.Errorf("container port must be in range 1-65535: %d", port)
		}
		p, ok := network.PortFrom(uint16(port), network.TCP)
		if !ok {
			return fmt.Errorf("invalid container port: %d", port)
		}
		cfg.containerPort = p
		return nil
	})
}

// WithHostIP sets the host IP to bind the container port to.
func WithHostIP(hostIP string) Option {
	return optionFunc(func(cfg *config) error {
		hostIP = strings.TrimSpace(hostIP)
		if hostIP == "" {
			return errors.New("host IP must not be empty")
		}
		cfg.hostIP = hostIP
		return nil
	})
}

// WithHostPort sets a fixed host port. Leave unset to auto-assign.
func WithHostPort(port int) Option {
	return optionFunc(func(cfg *config) error {
		if port <= 0 || port > 65535 {
			return fmt.Errorf("host port must be in range 1-65535: %d", port)
		}
		cfg.hostPort = port
		return nil
	})
}

// WithEnv appends a single environment variable.
func WithEnv(key, value string) Option {
	return optionFunc(func(cfg *config) error {
		key = strings.TrimSpace(key)
		if key == "" {
			return errors.New("env key must not be empty")
		}
		if strings.Contains(key, "=") {
			return fmt.Errorf("env key must not contain '=': %s", key)
		}
		cfg.envVars = append(cfg.envVars, key+"="+value)
		return nil
	})
}

// WithEnvVars appends environment variables in KEY=VALUE format.
func WithEnvVars(envVars []string) Option {
	return optionFunc(func(cfg *config) error {
		cfg.envVars = append(cfg.envVars, slices.Clone(envVars)...)
		return nil
	})
}

// WithAutoRemove configures Docker AutoRemove behavior.
func WithAutoRemove(autoRemove bool) Option {
	return optionFunc(func(cfg *config) error {
		cfg.autoRemove = autoRemove
		return nil
	})
}

// WithPullProgress sets where image pull output is streamed.
func WithPullProgress(w io.Writer) Option {
	return optionFunc(func(cfg *config) error {
		cfg.pullProgress = w
		return nil
	})
}

// WithLabel sets a single container label.
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
