package contact

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	pageSize = 10
	schema   = `
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
PRAGMA foreign_keys=ON;
CREATE TABLE IF NOT EXISTS contacts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	first_name TEXT NOT NULL DEFAULT '',
	last_name TEXT NOT NULL DEFAULT '',
	phone TEXT NOT NULL DEFAULT '',
	email TEXT NOT NULL UNIQUE COLLATE NOCASE
);
PRAGMA user_version = 1;
`
)

// Contact is one address-book row.
type Contact struct {
	ID     int64
	First  string
	Last   string
	Phone  string
	Email  string
	Errors map[string]string
}

// Page is one listing window and whether another window exists.
type Page struct {
	Contacts []Contact
	HasMore  bool
}

// InvalidError is a validation failure that should be shown on the form.
type InvalidError struct {
	Contact Contact
}

func (err *InvalidError) Error() string {
	return "contact is invalid"
}

// NotFoundError is a lookup failure for a missing contact id.
type NotFoundError struct {
	ID int64
}

func (err *NotFoundError) Error() string {
	return "contact not found"
}

// Store is a SQLite-backed contact repository.
type Store struct {
	db *sql.DB
}

// Open creates or opens a SQLite database and ensures the schema exists.
func Open(path string) (*Store, error) {
	directory := filepath.Dir(path)
	if directory != "." && directory != "" {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}
	database, _ := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	database.SetMaxOpenConns(1)
	store := &Store{db: database}
	if err := store.prepare(); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

// Close releases the database.
func (store *Store) Close() error {
	if store == nil || store.db == nil {
		return nil
	}
	return store.db.Close()
}

// Count returns the number of stored contacts.
func (store *Store) Count(ctx context.Context) (int, error) {
	var total int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM contacts`).Scan(&total); err != nil {
		return 0, fmt.Errorf("count contacts: %w", err)
	}
	return total, nil
}

// List returns one page of contacts, optionally filtered by a search term.
func (store *Store) List(ctx context.Context, query string, page int) (Page, error) {
	if page < 1 {
		page = 1
	}
	rows, err := store.queryPage(ctx, query, page)
	if err != nil {
		return Page{}, err
	}
	defer func() { _ = rows.Close() }()

	listed := Page{Contacts: make([]Contact, 0, pageSize)}
	for rows.Next() {
		var item Contact
		_ = rows.Scan(&item.ID, &item.First, &item.Last, &item.Phone, &item.Email)
		listed.Contacts = append(listed.Contacts, item)
	}
	if len(listed.Contacts) > pageSize {
		listed.HasMore = true
		listed.Contacts = listed.Contacts[:pageSize]
	}
	return listed, rows.Err()
}

// Find returns a contact by id.
func (store *Store) Find(ctx context.Context, id int64) (Contact, error) {
	item, err := scanContact(store.db.QueryRowContext(ctx, `
SELECT id, first_name, last_name, phone, email
FROM contacts
WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Contact{}, &NotFoundError{ID: id}
	}
	if err != nil {
		return Contact{}, fmt.Errorf("find contact %d: %w", id, err)
	}
	return item, nil
}

// Create inserts a contact after validation.
func (store *Store) Create(ctx context.Context, item Contact) (Contact, error) {
	if err := normalize(&item); err != nil {
		return Contact{}, err
	}
	err := store.db.QueryRowContext(ctx, `
INSERT INTO contacts (first_name, last_name, phone, email)
VALUES (?, ?, ?, ?)
RETURNING id`, item.First, item.Last, item.Phone, item.Email).Scan(&item.ID)
	return item, store.writeError(item, err, "create contact")
}

// Update writes a contact after validation.
func (store *Store) Update(ctx context.Context, item Contact) (Contact, error) {
	if err := normalize(&item); err != nil {
		return Contact{}, err
	}
	result, err := store.db.ExecContext(ctx, `
UPDATE contacts
SET first_name = ?, last_name = ?, phone = ?, email = ?
WHERE id = ?`, item.First, item.Last, item.Phone, item.Email, item.ID)
	if err := store.writeError(item, err, "update contact"); err != nil {
		return Contact{}, err
	}
	return item, requireAffected(result, item.ID)
}

// Delete removes a contact by id.
func (store *Store) Delete(ctx context.Context, id int64) error {
	result, err := store.db.ExecContext(ctx, `DELETE FROM contacts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete contact: %w", err)
	}
	return requireAffected(result, id)
}

// EmailError returns the email validation message for an existing contact.
func (store *Store) EmailError(ctx context.Context, id int64, email string) (string, error) {
	item, err := store.Find(ctx, id)
	if err != nil {
		return "", err
	}
	return store.CheckEmail(ctx, item.ID, email)
}

// CheckEmail returns the validation message for an email without loading the row.
func (store *Store) CheckEmail(ctx context.Context, id int64, email string) (string, error) {
	item := Contact{ID: id, Email: email}
	err := store.validateUnique(ctx, &item)
	var invalid *InvalidError
	if errors.As(err, &invalid) {
		return invalid.Contact.Error("email"), nil
	}
	return "", err
}

// Seed inserts contacts when the table is empty.
func (store *Store) Seed(ctx context.Context, contacts []Contact) error {
	total, err := store.Count(ctx)
	if err != nil || total > 0 {
		return err
	}
	for _, item := range contacts {
		if _, err := store.Create(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

// Error returns the validation message for a field.
func (item Contact) Error(field string) string {
	if item.Errors == nil {
		return ""
	}
	return item.Errors[field]
}

func (store *Store) prepare() error {
	if err := store.db.Ping(); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := store.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate contacts: %w", err)
	}
	return nil
}

func (store *Store) queryPage(ctx context.Context, query string, page int) (*sql.Rows, error) {
	offset := (page - 1) * pageSize
	limit := pageSize + 1
	if query == "" {
		rows, err := store.db.QueryContext(ctx, `
SELECT id, first_name, last_name, phone, email
FROM contacts
ORDER BY last_name, first_name, id
LIMIT ? OFFSET ?`, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("list contacts: %w", err)
		}
		return rows, nil
	}
	pattern := "%" + query + "%"
	rows, err := store.db.QueryContext(ctx, `
SELECT id, first_name, last_name, phone, email
FROM contacts
WHERE first_name LIKE ? COLLATE NOCASE
   OR last_name LIKE ? COLLATE NOCASE
   OR email LIKE ? COLLATE NOCASE
   OR phone LIKE ?
ORDER BY last_name, first_name, id
LIMIT ? OFFSET ?`, pattern, pattern, pattern, pattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}
	return rows, nil
}

func (store *Store) validateUnique(ctx context.Context, item *Contact) error {
	if err := normalize(item); err != nil {
		return err
	}
	var existing int64
	err := store.db.QueryRowContext(ctx, `
SELECT id FROM contacts WHERE email = ? COLLATE NOCASE AND id != ?`, item.Email, item.ID).Scan(&existing)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check email: %w", err)
	}
	item.Errors = map[string]string{"email": "Email Must Be Unique"}
	return &InvalidError{Contact: *item}
}

func (store *Store) writeError(item Contact, err error, op string) error {
	if err == nil {
		return nil
	}
	if uniqueViolation(err) {
		item.Errors = map[string]string{"email": "Email Must Be Unique"}
		return &InvalidError{Contact: item}
	}
	return fmt.Errorf("%s: %w", op, err)
}

func normalize(item *Contact) error {
	item.First = strings.TrimSpace(item.First)
	item.Last = strings.TrimSpace(item.Last)
	item.Phone = strings.TrimSpace(item.Phone)
	item.Email = strings.TrimSpace(item.Email)
	if item.Email != "" {
		return nil
	}
	item.Errors = map[string]string{"email": "Email Required"}
	return &InvalidError{Contact: *item}
}

func requireAffected(result sql.Result, id int64) error {
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return &NotFoundError{ID: id}
	}
	return nil
}

func uniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanContact(row rowScanner) (Contact, error) {
	var item Contact
	if err := row.Scan(&item.ID, &item.First, &item.Last, &item.Phone, &item.Email); err != nil {
		return Contact{}, err
	}
	return item, nil
}
