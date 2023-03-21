package goose

import "github.com/pressly/goose/v4/internal/migration"

/*

Unimplemented. this is a placeholder for a future feature.

See the following issues for more details:

 - https://github.com/pressly/goose/issues/222
 - https://github.com/pressly/goose/issues/485

*/

func splitMigrationsIntoGroups(migrations []*migration.Migration) [][]*migration.Migration {
	groups := make([][]*migration.Migration, 0)
	var prev bool
	for _, m := range migrations {
		if len(groups) == 0 {
			groups = append(groups, []*migration.Migration{m})
			prev = m.UseTx()
			continue
		}
		if prev && m.UseTx() {
			groups[len(groups)-1] = append(groups[len(groups)-1], m)
		} else {
			groups = append(groups, []*migration.Migration{m})
		}
		prev = m.UseTx()
	}
	return groups
}
