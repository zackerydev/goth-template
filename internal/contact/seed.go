package contact

import "context"

// SeedIfEmpty inserts the demo address book when the table is empty.
func (store *Store) SeedIfEmpty(ctx context.Context) error {
	return store.Seed(ctx, demoContacts())
}

func demoContacts() []Contact {
	rows := [][4]string{
		{"Joe", "Blow", "702-555-0101", "joe@blow.com"},
		{"Sarah", "Connor", "213-555-0144", "sarah@resistance.org"},
		{"Miles", "Dyson", "310-555-0188", "miles@cyberdyne.com"},
		{"Ada", "Lovelace", "020-555-0112", "ada@analytical.engine"},
		{"Alan", "Turing", "0161-555-0199", "alan@bletchley.uk"},
		{"Grace", "Hopper", "202-555-0133", "grace@cobol.dev"},
		{"Barbara", "Liskov", "617-555-0165", "barbara@mit.edu"},
		{"Donald", "Knuth", "650-555-0120", "don@artofprogramming.com"},
		{"Margaret", "Hamilton", "281-555-0177", "margaret@apollo.nasa"},
		{"Ken", "Thompson", "908-555-0108", "ken@bell-labs.com"},
		{"Dennis", "Ritchie", "908-555-0109", "dmr@bell-labs.com"},
		{"Rob", "Pike", "415-555-0155", "rob@swtch.com"},
		{"Carin", "Meier", "614-555-0121", "carin@clojure.org"},
		{"Leslie", "Lamport", "650-555-0182", "leslie@tex.dev"},
		{"Hedy", "Lamarr", "323-555-0142", "hedy@spread.spectrum"},
	}
	contacts := make([]Contact, 0, len(rows))
	for _, row := range rows {
		contacts = append(contacts, Contact{First: row[0], Last: row[1], Phone: row[2], Email: row[3]})
	}
	return contacts
}
