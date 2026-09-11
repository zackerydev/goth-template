package contact_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zackerydev/goth-template/internal/contact"
)

func TestStoreCRUDSearchAndSeed(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	ctx := context.Background()

	if err := store.SeedIfEmpty(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := store.SeedIfEmpty(ctx); err != nil {
		t.Fatalf("seed twice: %v", err)
	}
	total, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 15 {
		t.Fatalf("count = %d, want 15", total)
	}

	page, err := store.List(ctx, "", 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Contacts) != 10 || !page.HasMore {
		t.Fatalf("first page = %d hasMore=%t", len(page.Contacts), page.HasMore)
	}
	second, err := store.List(ctx, "", 0)
	if err != nil {
		t.Fatalf("list page 0: %v", err)
	}
	if second.Contacts[0].ID != page.Contacts[0].ID {
		t.Fatal("page 0 did not clamp to page 1")
	}
	next, err := store.List(ctx, "", 2)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(next.Contacts) != 5 || next.HasMore {
		t.Fatalf("second page = %d hasMore=%t", len(next.Contacts), next.HasMore)
	}

	found, err := store.List(ctx, "liskov", 1)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found.Contacts) != 1 || found.Contacts[0].Last != "Liskov" {
		t.Fatalf("search = %#v", found.Contacts)
	}
}

func TestStoreCreateUpdateDelete(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	ctx := context.Background()

	created, err := store.Create(ctx, contact.Contact{
		First: "Carson",
		Last:  "Gross",
		Phone: "555-0100",
		Email: "carson@htmx.org",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	loaded, err := store.Find(ctx, created.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if loaded.Email != "carson@htmx.org" {
		t.Fatalf("loaded email = %q", loaded.Email)
	}

	loaded.Phone = "555-0199"
	updated, err := store.Update(ctx, loaded)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Phone != "555-0199" {
		t.Fatalf("updated phone = %q", updated.Phone)
	}

	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Find(ctx, created.ID); !notFound(err) {
		t.Fatalf("find deleted = %v", err)
	}
}

func TestStoreValidation(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	ctx := context.Background()
	first, err := store.Create(ctx, contact.Contact{First: "Ada", Email: "ada@example.com"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = store.Create(ctx, contact.Contact{Email: ""})
	if !invalid(err, "Email Required") {
		t.Fatalf("empty email = %v", err)
	}
	_, err = store.Create(ctx, contact.Contact{Email: "ADA@example.com"})
	if !invalid(err, "Email Must Be Unique") {
		t.Fatalf("duplicate email = %v", err)
	}

	message, err := store.EmailError(ctx, first.ID, "ada@example.com")
	if err != nil || message != "" {
		t.Fatalf("same email = %q %v", message, err)
	}
	message, err = store.EmailError(ctx, first.ID, "")
	if err != nil || message != "Email Required" {
		t.Fatalf("required email = %q %v", message, err)
	}
	second, err := store.Create(ctx, contact.Contact{Email: "other@example.com"})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	message, err = store.EmailError(ctx, first.ID, second.Email)
	if err != nil || message != "Email Must Be Unique" {
		t.Fatalf("taken email = %q %v", message, err)
	}

	if _, err := store.Update(ctx, contact.Contact{ID: first.ID}); !invalid(err, "Email Required") {
		t.Fatalf("update empty email = %v", err)
	}
	first.Email = second.Email
	if _, err := store.Update(ctx, first); !invalid(err, "Email Must Be Unique") {
		t.Fatalf("update duplicate = %v", err)
	}
	if _, err := store.Update(ctx, contact.Contact{ID: 999, Email: "x@y.z"}); !notFound(err) {
		t.Fatalf("update missing = %v", err)
	}
	if err := store.Delete(ctx, 999); !notFound(err) {
		t.Fatalf("delete missing = %v", err)
	}
	if _, err := store.EmailError(ctx, 999, "x@y.z"); !notFound(err) {
		t.Fatalf("email missing = %v", err)
	}
	if got := (&contact.InvalidError{}).Error(); got != "contact is invalid" {
		t.Fatalf("invalid error = %q", got)
	}
	if got := (&contact.NotFoundError{ID: 9}).Error(); got != "contact not found" {
		t.Fatalf("not found error = %q", got)
	}
}

func TestStoreInitializationAndClosedErrors(t *testing.T) {
	t.Parallel()

	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("nope"), 0o600); err != nil {
		t.Fatalf("write blocked path: %v", err)
	}
	if _, err := contact.Open(filepath.Join(blocked, "contacts.db")); err == nil {
		t.Fatal("open with file parent succeeded")
	}

	garbage := filepath.Join(t.TempDir(), "garbage.db")
	if err := os.WriteFile(garbage, []byte("not sqlite"), 0o600); err != nil {
		t.Fatalf("write garbage: %v", err)
	}
	if _, err := contact.Open(garbage); err == nil {
		t.Fatal("open garbage database succeeded")
	}

	if err := (*contact.Store)(nil).Close(); err != nil {
		t.Fatalf("nil close: %v", err)
	}
	if err := (&contact.Store{}).Close(); err != nil {
		t.Fatalf("empty close: %v", err)
	}

	store := openStore(t)
	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	ctx := context.Background()
	if _, err := store.Count(ctx); err == nil {
		t.Fatal("count on closed store succeeded")
	}
	if _, err := store.List(ctx, "", 1); err == nil {
		t.Fatal("list on closed store succeeded")
	}
	if _, err := store.List(ctx, "ada", 1); err == nil {
		t.Fatal("search on closed store succeeded")
	}
	if _, err := store.Find(ctx, 1); err == nil {
		t.Fatal("find on closed store succeeded")
	}
	if _, err := store.Create(ctx, contact.Contact{Email: "a@b.c"}); err == nil {
		t.Fatal("create on closed store succeeded")
	}
	if err := store.SeedIfEmpty(ctx); err == nil {
		t.Fatal("seed on closed store succeeded")
	}
}

func TestCanceledContextAndBadSeed(t *testing.T) {
	t.Parallel()

	live := openStore(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := live.Create(canceled, contact.Contact{Email: "cancel@example.com"}); err == nil {
		t.Fatal("create with canceled context succeeded")
	}
	if _, err := live.List(canceled, "", 1); err == nil {
		t.Fatal("list with canceled context succeeded")
	}
	if _, err := live.List(canceled, "ada", 1); err == nil {
		t.Fatal("search with canceled context succeeded")
	}
	if _, err := live.Find(canceled, 1); err == nil {
		t.Fatal("find with canceled context succeeded")
	}
	if err := live.Delete(canceled, 1); err == nil {
		t.Fatal("delete with canceled context succeeded")
	}
	if _, err := live.Update(canceled, contact.Contact{ID: 1, Email: "cancel@example.com"}); err == nil {
		t.Fatal("update with canceled context succeeded")
	}
	if _, err := live.EmailError(canceled, 1, "cancel@example.com"); err == nil {
		t.Fatal("email with canceled context succeeded")
	}
	if _, err := live.CheckEmail(canceled, 1, "cancel@example.com"); err == nil {
		t.Fatal("check email with canceled context succeeded")
	}
	if err := live.Seed(context.Background(), []contact.Contact{{Email: ""}}); err == nil {
		t.Fatal("seed with invalid contact succeeded")
	}
}

func TestOpenFailures(t *testing.T) {
	t.Parallel()

	if _, err := contact.Open(t.TempDir()); err == nil {
		t.Fatal("open directory as database succeeded")
	}

	path := filepath.Join(t.TempDir(), "readonly.db")
	ready, err := contact.Open(path)
	if err != nil {
		t.Fatalf("open writable database: %v", err)
	}
	if err := ready.Close(); err != nil {
		t.Fatalf("close writable database: %v", err)
	}
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatalf("chmod database: %v", err)
	}
	_ = os.Chmod(path+"-wal", 0o400)
	_ = os.Chmod(path+"-shm", 0o400)
	if _, err := contact.Open(path); err == nil {
		t.Fatal("open read-only database succeeded")
	}
}

func openStore(t *testing.T) *contact.Store {
	t.Helper()
	store, err := contact.Open(filepath.Join(t.TempDir(), "contacts.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func invalid(err error, message string) bool {
	var invalidErr *contact.InvalidError
	return errors.As(err, &invalidErr) && invalidErr.Contact.Error("email") == message
}

func notFound(err error) bool {
	var missing *contact.NotFoundError
	return errors.As(err, &missing)
}
