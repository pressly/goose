package dockermanage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/netip"
	"os"
	"strconv"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/sethvargo/go-retry"
)

const (
	defaultReadinessTimeout = 30 * time.Second
	defaultReadinessDelay   = 500 * time.Millisecond
)

// Container is a running Docker container managed by this package.
type Container struct {
	ID     string
	Image  string
	Host   string
	Port   int
	Labels map[string]string
}

// ReadinessFunc reports whether a container is ready.
type ReadinessFunc func(ctx context.Context, container *Container) error

// Manager manages Docker containers using the native Docker client.
type Manager struct {
	client *client.Client
	logger *slog.Logger
}

// NewManager creates a new manager backed by the Docker client configured from environment.
func NewManager(logger *slog.Logger) (*Manager, error) {
	dockerClient, err := client.New(
		client.FromEnv,
	)
	if err != nil {
		return nil, fmt.Errorf("create Docker client: %w", err)
	}
	return newManagerWithClient(dockerClient, logger), nil
}

func newManagerWithClient(dockerClient *client.Client, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Manager{
		client: dockerClient,
		logger: logger.With(slog.String("logger", "dockermanage")),
	}
}

// Start starts a container with the provided options.
func (m *Manager) Start(ctx context.Context, options ...Option) (_ *Container, retErr error) {
	cfg := defaultConfig()
	cfg.pullProgress = os.Stderr
	for _, opt := range options {
		if opt == nil {
			continue
		}
		if err := opt.apply(cfg); err != nil {
			return nil, err
		}
	}
	if cfg.image == "" {
		return nil, errors.New("image is required")
	}
	if cfg.containerPort.IsZero() {
		return nil, errors.New("container port is required")
	}
	if err := m.pullImageIfNotExists(ctx, cfg.image, cfg.pullProgress); err != nil {
		return nil, fmt.Errorf("pull image %s: %w", cfg.image, err)
	}

	portBinding := network.PortBinding{HostIP: netip.MustParseAddr(cfg.hostIP)}
	if cfg.hostPort > 0 {
		portBinding.HostPort = strconv.Itoa(cfg.hostPort)
	}

	resp, err := m.client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name: cfg.name,
		Config: &container.Config{
			Image: cfg.image,
			Env:   cfg.envVars,
			ExposedPorts: network.PortSet{
				cfg.containerPort: struct{}{},
			},
			Labels: maps.Clone(cfg.labels),
		},
		HostConfig: &container.HostConfig{
			PortBindings: network.PortMap{cfg.containerPort: []network.PortBinding{portBinding}},
			AutoRemove:   cfg.autoRemove,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}
	defer func() {
		if retErr != nil {
			cleanupCtx := context.WithoutCancel(ctx)
			_, err := m.client.ContainerRemove(cleanupCtx, resp.ID, client.ContainerRemoveOptions{Force: true})
			if err != nil {
				m.logger.Error(
					"remove container after start failure",
					slog.String("container_id", resp.ID),
					slog.Any("error", err),
				)
			}
		}
	}()

	if _, err := m.client.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}

	hostPort := cfg.hostPort
	if hostPort == 0 {
		inspectResult, err := m.client.ContainerInspect(ctx, resp.ID, client.ContainerInspectOptions{})
		if err != nil {
			return nil, fmt.Errorf("inspect container for port: %w", err)
		}
		hostPort, err = resolveBoundPort(inspectResult.Container, cfg.containerPort)
		if err != nil {
			return nil, fmt.Errorf("resolve host port: %w", err)
		}
	}

	m.logger.Info(
		"docker container started",
		slog.String("container_id", resp.ID),
		slog.String("image", cfg.image),
		slog.Int("port", hostPort),
	)
	return &Container{
		ID:     resp.ID,
		Image:  cfg.image,
		Host:   cfg.hostIP,
		Port:   hostPort,
		Labels: maps.Clone(cfg.labels),
	}, nil
}

func resolveBoundPort(containerJSON container.InspectResponse, containerPort network.Port) (int, error) {
	if containerJSON.NetworkSettings == nil {
		return 0, errors.New("container network settings are missing")
	}
	portBindings, ok := containerJSON.NetworkSettings.Ports[containerPort]
	if !ok || len(portBindings) == 0 {
		return 0, fmt.Errorf("no port bindings found for %s", containerPort)
	}
	for _, binding := range portBindings {
		if binding.HostPort == "" {
			continue
		}
		port, err := strconv.Atoi(binding.HostPort)
		if err != nil {
			return 0, fmt.Errorf("parse host port %q: %w", binding.HostPort, err)
		}
		return port, nil
	}
	return 0, fmt.Errorf("no host port found for %s", containerPort)
}

// Stop stops a running container.
func (m *Manager) Stop(ctx context.Context, containerID string) error {
	if _, err := m.client.ContainerStop(ctx, containerID, client.ContainerStopOptions{}); err != nil {
		return fmt.Errorf("stop container %s: %w", containerID, err)
	}
	m.logger.Info("docker container stopped", slog.String("container_id", containerID))
	return nil
}

// Remove removes a container. If running, it is force removed.
func (m *Manager) Remove(ctx context.Context, containerID string) error {
	if _, err := m.client.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("remove container %s: %w", containerID, err)
	}
	m.logger.Info("docker container removed", slog.String("container_id", containerID))
	return nil
}

// WaitOption configures WaitReady behavior.
type WaitOption func(*waitConfig)

type waitConfig struct {
	timeout time.Duration
	delay   time.Duration
}

// WithTimeout sets the maximum time to wait for readiness. Defaults to 30s.
func WithTimeout(d time.Duration) WaitOption {
	return func(cfg *waitConfig) { cfg.timeout = d }
}

// WithDelay sets the interval between readiness checks. Defaults to 500ms.
func WithDelay(d time.Duration) WaitOption {
	return func(cfg *waitConfig) { cfg.delay = d }
}

// WaitReady waits until a custom readiness checker succeeds.
func (m *Manager) WaitReady(ctx context.Context, container *Container, readiness ReadinessFunc, opts ...WaitOption) error {
	if container == nil {
		return errors.New("container must not be nil")
	}
	if readiness == nil {
		return errors.New("readiness function must not be nil")
	}

	cfg := &waitConfig{
		timeout: defaultReadinessTimeout,
		delay:   defaultReadinessDelay,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.timeout <= 0 {
		return fmt.Errorf("timeout must be positive: %v", cfg.timeout)
	}

	retryCtx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	backoff := retry.NewConstant(cfg.delay)
	err := retry.Do(retryCtx, backoff, func(ctx context.Context) error {
		if err := readiness(ctx, container); err != nil {
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("container %s did not become ready within %s: %w", container.ID, cfg.timeout, err)
	}
	m.logger.Info("docker container ready", slog.String("container_id", container.ID))
	return nil
}

// ListManaged returns all container IDs started by this package.
func (m *Manager) ListManaged(ctx context.Context) ([]string, error) {
	result, err := m.client.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: client.Filters{}.Add("label", ManagedLabelKey),
	})
	if err != nil {
		return nil, fmt.Errorf("list managed containers: %w", err)
	}
	ids := make([]string, 0, len(result.Items))
	for _, c := range result.Items {
		ids = append(ids, c.ID)
	}
	return ids, nil
}

// StopManaged stops all containers started by this package.
func (m *Manager) StopManaged(ctx context.Context) error {
	ids, err := m.ListManaged(ctx)
	if err != nil {
		return fmt.Errorf("list containers for stop: %w", err)
	}
	var errs []error
	for _, id := range ids {
		if err := m.Stop(ctx, id); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	m.logger.Info("stopped all managed containers", slog.Int("count", len(ids)))
	return nil
}

// RemoveManaged removes all containers started by this package.
func (m *Manager) RemoveManaged(ctx context.Context) error {
	ids, err := m.ListManaged(ctx)
	if err != nil {
		return fmt.Errorf("list containers for remove: %w", err)
	}
	var errs []error
	for _, id := range ids {
		if err := m.Remove(ctx, id); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	m.logger.Info("removed all managed containers", slog.Int("count", len(ids)))
	return nil
}

// ExecOptions configures a command to run inside a container.
type ExecOptions struct {
	Cmd    []string  // Command and arguments
	Env    []string  // Optional environment variables
	Stdout io.Writer // Where to write stdout (nil → discard)
	Stderr io.Writer // Where to write stderr (nil → discard)
}

// ExecResult holds the outcome of an exec invocation.
type ExecResult struct {
	ExitCode int
}

// Exec runs a command inside a running container and streams its output.
func (m *Manager) Exec(ctx context.Context, containerID string, opts ExecOptions) (*ExecResult, error) {
	if len(opts.Cmd) == 0 {
		return nil, errors.New("cmd must not be empty")
	}
	stdout := opts.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	createResp, err := m.client.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		Cmd:          opts.Cmd,
		Env:          opts.Env,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("exec create: %w", err)
	}

	attachResp, err := m.client.ExecAttach(ctx, createResp.ID, client.ExecAttachOptions{})
	if err != nil {
		return nil, fmt.Errorf("exec attach: %w", err)
	}
	defer attachResp.Close()

	if err := demuxDockerStream(attachResp.Reader, stdout, stderr); err != nil {
		return nil, fmt.Errorf("exec stream: %w", err)
	}

	inspectResp, err := m.client.ExecInspect(ctx, createResp.ID, client.ExecInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("exec inspect: %w", err)
	}

	m.logger.Info(
		"docker exec completed",
		slog.String("container_id", containerID),
		slog.Int("exit_code", inspectResp.ExitCode),
	)
	return &ExecResult{ExitCode: inspectResp.ExitCode}, nil
}

// Close closes the underlying Docker client.
func (m *Manager) Close() error {
	return m.client.Close()
}

// Docker multiplexed stream constants. When TTY is disabled, Docker prefixes each output frame with
// an 8-byte header: [streamType(1), padding(3), payloadSize(4 big-endian)].
const (
	streamStdout     byte = 1
	streamStderr     byte = 2
	streamHeaderSize      = 8
)

// demuxDockerStream reads a Docker multiplexed stream and routes each frame to the appropriate
// writer based on its stream type.
func demuxDockerStream(r io.Reader, stdout, stderr io.Writer) error {
	var header [streamHeaderSize]byte
	for {
		if _, err := io.ReadFull(r, header[:]); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		payloadSize := int64(binary.BigEndian.Uint32(header[4:]))
		var dst io.Writer
		switch header[0] {
		case streamStdout:
			dst = stdout
		case streamStderr:
			dst = stderr
		default:
			dst = io.Discard
		}
		if _, err := io.CopyN(dst, r, payloadSize); err != nil {
			return err
		}
	}
}

func (m *Manager) pullImageIfNotExists(ctx context.Context, imageName string, progressWriter io.Writer) (retErr error) {
	if _, err := m.client.ImageInspect(ctx, imageName); err == nil {
		return nil
	} else if !errdefs.IsNotFound(err) {
		return fmt.Errorf("inspect image: %w", err)
	}

	reader, err := m.client.ImagePull(ctx, imageName, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull image: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, reader.Close())
	}()

	if progressWriter == nil {
		progressWriter = io.Discard
	}
	if _, err := io.Copy(progressWriter, reader); err != nil {
		return fmt.Errorf("stream pull output: %w", err)
	}
	return nil
}
