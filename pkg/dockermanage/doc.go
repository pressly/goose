// Package dockermanage provides lightweight Docker container lifecycle helpers for integration
// testing.
//
// A [Manager] wraps the native Docker client and exposes methods to start, stop, remove, and list
// containers. Containers are configured through functional [Option] values such as [WithImage],
// [WithContainerPortTCP], and [WithEnv].
//
// After starting a container, use [Manager.WaitReady] with a custom [ReadinessFunc] to block until
// the service inside the container is accepting connections.
//
// Every container created through this package is tagged with the [ManagedLabelKey] label, which
// allows bulk operations like [Manager.StopManaged] and [Manager.RemoveManaged] to clean up all
// managed containers.
//
// Database-specific sub-packages (e.g., postgres) provide opinionated defaults for common
// databases.
package dockermanage
