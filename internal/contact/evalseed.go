package contact

import "context"

// SeedEvalIfEmpty inserts the agent-eval address book when the table is empty.
func (store *Store) SeedEvalIfEmpty(ctx context.Context) error {
	return store.Seed(ctx, EvalContacts())
}

// EvalContacts is the isolated eval fixture: Street Fighter names, several Ryus,
// Dan with a last name that is not Hibiki, Chun-Li present, and no Ken Masters.
func EvalContacts() []Contact {
	rows := [][4]string{
		{"Dan", "Smith", "555-0101", "dan@eval.test"},
		{"Chun-Li", "Xiang", "555-0102", "chun-li@eval.test"},
		{"Ryu", "Dojo", "555-0103", "ryu@eval.test"},
		{"Kenji", "Ryu", "555-0104", "kenji.ryu@eval.test"},
		{"Ryu", "Junior", "555-0105", "ryu.jr@eval.test"},
		{"Guile", "USA", "555-0106", "guile@eval.test"},
		{"Cammy", "White", "555-0107", "cammy@eval.test"},
		{"Zangief", "Red", "555-0108", "zangief@eval.test"},
		{"Dhalsim", "Yoga", "555-0109", "dhalsim@eval.test"},
		{"Sagat", "Tiger", "555-0110", "sagat@eval.test"},
		{"Blanka", "Green", "555-0111", "blanka@eval.test"},
		{"Edmond", "Honda", "555-0112", "honda@eval.test"},
		{"Vega", "Spain", "555-0113", "vega@eval.test"},
		{"Akuma", "Demon", "555-0114", "akuma@eval.test"},
		{"Sakura", "Kasugano", "555-0115", "sakura@eval.test"},
		{"Juri", "Han", "555-0116", "juri@eval.test"},
	}
	contacts := make([]Contact, 0, len(rows))
	for _, row := range rows {
		contacts = append(contacts, Contact{First: row[0], Last: row[1], Phone: row[2], Email: row[3]})
	}
	return contacts
}
