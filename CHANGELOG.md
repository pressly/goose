# Changelog

## [Unreleased]

## [v3.13.1] - 2023-07-03

- Add pre-built binaries with GoReleaser and update the build process.

## [v3.13.0] - 2023-06-29

- Update go.mod and retract all v3.12.X tags. They were accidentally pushed and contain a reference
  to the wrong Go module.
- Fix `up` and `up -allowing-missing` behavior.
- Fix empty version in log output.
- Add new `context.Context`-aware functions and methods, for both sql and go migrations.
- Return error when no migration files found or dir is not a directory.

[Unreleased]: https://github.com/pressly/goose/compare/v3.13.1...HEAD
[v3.13.1]: https://github.com/pressly/goose/compare/v3.13.0...v3.13.1
[v3.13.0]: https://github.com/pressly/goose/compare/v3.11.2...v3.13.0
