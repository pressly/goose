package migration

type VersionID = int64

// Version TODO(mf): in the future, we maybe want to expand this struct so implementors can store
// additional information. See the following issues for more information:
//   - https://github.com/pressly/goose/issues/422
//   - https://github.com/pressly/goose/issues/288
type Version interface {
	GetID() VersionID
}

const (
	ZeroVersionID VersionID = 0
	NoVersionID   VersionID = -1
)

var (
	ZeroVersion = &version{id: ZeroVersionID}
	NoVersion   = &version{id: NoVersionID}
)

func NewVersion(id VersionID) Version {
	return &version{id: id}
}

type version struct {
	id VersionID
}

func (v *version) GetID() VersionID { return v.id }
