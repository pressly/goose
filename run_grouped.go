package goose

/*

Unimplemented. this is a placeholder for a future feature.

See the following issues for more details:

 - https://github.com/pressly/goose/issues/222
 - https://github.com/pressly/goose/issues/485

*/

func splitMigrationsIntoGroups(migrations []*migration) [][]*migration {
	groups := make([][]*migration, 0)
	var prev bool
	for _, m := range migrations {
		if len(groups) == 0 {
			groups = append(groups, []*migration{m})
			prev = m.useTx()
			continue
		}
		if prev && m.useTx() {
			groups[len(groups)-1] = append(groups[len(groups)-1], m)
		} else {
			groups = append(groups, []*migration{m})
		}
		prev = m.useTx()
	}
	return groups
}
