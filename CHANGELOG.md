# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v3.18.0] - 2024-01-31

- Add environment variable substitution for SQL migrations. (#604)

  - This feature is **disabled by default**, and can be enabled by adding an annotation to the
    migration file:

    ```sql
    -- +goose ENVSUB ON
    ```

  - When enabled, goose will attempt to substitute environment variables in the SQL migration
    queries until the end of the file, or until the annotation `-- +goose ENVSUB OFF` is found. For
    example, if the environment variable `REGION` is set to `us_east_1`, the following SQL migration
    will be substituted to `SELECT * FROM regions WHERE name = 'us_east_1';`

    ```sql
    -- +goose ENVSUB ON
    -- +goose Up
    SELECT * FROM regions WHERE name = '${REGION}';
    ```

- Add native [Turso](https://turso.tech/) support with libsql driver. (#658)

- Fixed query for list migrations in YDB (#684)

## [v3.17.0] - 2023-12-15

- Standardised the MIT license (#647)
- Improve provider `Apply()` errors, add `ErrNotApplied` when attempting to rollback a migration
  that has not been previously applied. (#660)
- Add `WithDisableGlobalRegistry` option to `NewProvider` to disable the global registry. (#645)
- Add `-timeout` flag to CLI to set the maximum allowed duration for queries to run. Default remains
  no timeout. (#627)
- Add optional logging in `Provider` when `WithVerbose` option is supplied. (#668)

⚠️ Potential Breaking Change ⚠️

- Update `goose create` to use UTC time instead of local time. (#242)

## [v3.16.0] - 2023-11-12

- Added YDB support. (#592)
- Fix sqlserver query to ensure DB version. (#601)
- Allow setting / resetting the global Go migration registry. (#602)
  - `SetGlobalMigrations` and `ResetGlobalMigrations` functions have been added.
  - Introduce `NewGoMigration` for constructing Go migrations.
- Add initial implementation of `goose.NewProvider`.

🎉 Read more about this new feature here:

https://pressly.github.io/goose/blog/2023/goose-provider/

The motivation behind the Provider was simple - to reduce global state and make goose easier to
consume as an imported package.

Here's a quick summary:

- Avoid global state
- Make Provider safe to use concurrently
- Unlock (no pun intended) new features, such as database locking
- Make logging configurable
- Better error handling with proper return values
- Double down on Go migrations
- ... and more!

## [v3.15.1] - 2023-10-10

- Fix regression that prevented registering Go migrations that didn't have the corresponding files
  available in the filesystem. (#588)
  - If Go migrations have been registered globally, but there are no .go files in the filesystem,
    **always include** them.
  - If Go migrations have been registered, and there are .go files in the filesystem, **only
    include** those migrations. This was the original motivation behind #553.
  - If there are .go files in the filesystem but not registered, **raise an error**. This is to
    prevent accidentally adding valid looking Go migration files without explicitly registering
    them.

## [v3.15.0] - 2023-08-12

- Fix `sqlparser` to avoid skipping the last statement when it's not terminated with a semicolon
  within a StatementBegin/End block. (#580)
- Add `go1.21` to the CI matrix.
- Bump minimum version of module in go.mod to `go1.19`.
- Fix version output when installing pre-built binaries (#585).

## [v3.14.0] - 2023-07-26

- Filter registered Go migrations from the global map with corresponding .go files from the
  filesystem.
  - The code previously assumed all .go migrations would be in the same folder, so this should not
    be a breaking change.
  - See #553 for more details
- Improve output log message for applied up migrations. #562
- Fix an issue where `AddMigrationNoTxContext` was registering the wrong source because it skipped
  too many frames. #572
- Improve binary version output when using go install.

## [v3.13.4] - 2023-07-07

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

[Unreleased]: https://github.com/pressly/goose/compare/v3.18.0...HEAD
[v3.18.0]: https://github.com/pressly/goose/compare/v3.17.0...v3.18.0
[v3.17.0]: https://github.com/pressly/goose/compare/v3.16.0...v3.17.0
[v3.16.0]: https://github.com/pressly/goose/compare/v3.15.1...v3.16.0
[v3.15.1]: https://github.com/pressly/goose/compare/v3.15.0...v3.15.1
[v3.15.0]: https://github.com/pressly/goose/compare/v3.14.0...v3.15.0
[v3.14.0]: https://github.com/pressly/goose/compare/v3.13.4...v3.14.0
[v3.13.4]: https://github.com/pressly/goose/compare/v3.13.1...v3.13.4
[v3.13.1]: https://github.com/pressly/goose/compare/v3.13.0...v3.13.1
[v3.13.0]: https://github.com/pressly/goose/releases/tag/v3.13.0
