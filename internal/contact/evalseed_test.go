package contact_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/zackerydev/goth-template/internal/contact"
)

func TestEvalSeedFixture(t *testing.T) {
	t.Parallel()

	store, err := contact.Open(filepath.Join(t.TempDir(), "eval.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.SeedEvalIfEmpty(ctx); err != nil {
		t.Fatalf("seed eval: %v", err)
	}
	if err := store.SeedEvalIfEmpty(ctx); err != nil {
		t.Fatalf("seed eval twice: %v", err)
	}

	total, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 16 {
		t.Fatalf("count = %d, want 16", total)
	}
	if len(contact.EvalContacts()) != 16 {
		t.Fatalf("eval contacts = %d, want 16", len(contact.EvalContacts()))
	}

	ryu, err := store.CountNamed(ctx, "Ryu")
	if err != nil {
		t.Fatalf("count Ryu: %v", err)
	}
	if ryu != 3 {
		t.Fatalf("Ryu count = %d, want 3", ryu)
	}

	dan, err := store.List(ctx, "Dan", 1)
	if err != nil || len(dan.Contacts) != 1 || dan.Contacts[0].Last == "Hibiki" {
		t.Fatalf("Dan = %#v %v", dan.Contacts, err)
	}
	chun, err := store.List(ctx, "Chun-Li", 1)
	if err != nil || len(chun.Contacts) != 1 {
		t.Fatalf("Chun-Li = %#v %v", chun.Contacts, err)
	}
	ken, err := store.List(ctx, "Ken Masters", 1)
	if err != nil || len(ken.Contacts) != 0 {
		t.Fatalf("Ken Masters present: %#v %v", ken.Contacts, err)
	}

	page, err := store.List(ctx, "", 1)
	if err != nil || len(page.Contacts) != 10 || !page.HasMore {
		t.Fatalf("eval page 1 = %#v %v", page, err)
	}
}

func TestCountNamedErrors(t *testing.T) {
	t.Parallel()

	store, err := contact.Open(filepath.Join(t.TempDir(), "eval.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := store.CountNamed(context.Background(), "Ryu"); err == nil {
		t.Fatal("count named on closed store succeeded")
	}
	if err := store.SeedEvalIfEmpty(context.Background()); err == nil {
		t.Fatal("eval seed on closed store succeeded")
	}

	live, err := contact.Open(filepath.Join(t.TempDir(), "live.db"))
	if err != nil {
		t.Fatalf("open live: %v", err)
	}
	t.Cleanup(func() { _ = live.Close() })
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := live.CountNamed(canceled, "Ryu"); err == nil {
		t.Fatal("count named with canceled context succeeded")
	}
	if err := live.SeedEvalIfEmpty(canceled); err == nil {
		t.Fatal("eval seed with canceled context succeeded")
	}
}
