# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v3.13.2] - 2023-07-07

- Fix pre-built binary versioning and make small improvements to GoReleaser config.
- Fix an edge case in the `sqlparser` where the last up statement may be ignored if it's
  unterminated with a semicolon and followed by a `-- +goose Down` annotation.
- Trim `Logger` interface to `Printf` and `Fatalf` methods only. Projects that have previously
  implemented the `Logger` interface should not be affected, and can remove unused methods.

## [v3.13.1] - 2023-07-03

- Add pre-built binaries with GoReleaser and update the build process.

## [v3.13.0] - 2023-06-29

- Add a changelog to the project, based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
- Update go.mod and retract all `v3.12.X` tags. They were accidentally pushed and contain a
  reference to the wrong Go module.
- Fix `up` and `up -allowing-missing` behavior.
- Fix empty version in log output.
- Add new `context.Context`-aware functions and methods, for both sql and go migrations.
- Return error when no migration files found or dir is not a directory.

[Unreleased]: https://github.com/pressly/goose/compare/v3.13.2...HEAD
[v3.13.2]: https://github.com/pressly/goose/compare/v3.13.1...v3.13.2
[v3.13.1]: https://github.com/pressly/goose/compare/v3.13.0...v3.13.1
[v3.13.0]: https://github.com/pressly/goose/releases/tag/v3.13.0
