# AGENTS.md

## Go dependencies

- Upgrade with `make upgrade-deps`, not `go get -u ./...`. The script keeps pinned modules at their pinned versions.
- Check pins with `make check-deps`.
- Pins live in `PINS` in `scripts/go-deps.sh`. When you add one, also add it to the `ignore` list in `.github/dependabot.yaml`.

## GitHub Actions

- Pin actions to a full commit SHA with the tag in a comment: `uses: actions/setup-go@<sha> # v7.1.0`.
