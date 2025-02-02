package migration

type VersionID = int64

// Entity TODO(mf): in the future, we maybe want to expand this struct so implementors can store
// additional information. See the following issues for more information:
//   - https://github.com/pressly/goose/issues/422
//   - https://github.com/pressly/goose/issues/288
type Entity struct {
	Version VersionID
}
