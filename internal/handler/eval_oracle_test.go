package handler_test

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/zackerydev/goth-template/internal/contact"
)

func TestEvalOracleUpdateDan(t *testing.T) {
	t.Parallel()

	store, application := openEvalApp(t)
	dan := lookupNamed(t, store, "Dan")
	response := post(t, application, "/contacts/"+strconv.FormatInt(dan.ID, 10)+"/edit", url.Values{
		"first_name": {"Dan"},
		"last_name":  {"Hibiki"},
		"phone":      {dan.Phone},
		"email":      {dan.Email},
	})
	assertStatus(t, response, http.StatusSeeOther)
	updated := lookupNamed(t, store, "Dan")
	if updated.Last != "Hibiki" {
		t.Fatalf("Dan last = %q, want Hibiki", updated.Last)
	}
}

func TestEvalOracleCreateKen(t *testing.T) {
	t.Parallel()

	store, application := openEvalApp(t)
	response := post(t, application, "/contacts/new", url.Values{
		"first_name": {"Ken"},
		"last_name":  {"Masters"},
		"phone":      {"555-0199"},
		"email":      {"ken.masters@eval.test"},
	})
	assertStatus(t, response, http.StatusSeeOther)
	found, err := store.List(context.Background(), "Masters", 1)
	if err != nil || len(found.Contacts) != 1 || found.Contacts[0].First != "Ken" {
		t.Fatalf("Ken Masters = %#v %v", found.Contacts, err)
	}
}

func TestEvalOracleDeleteChunLi(t *testing.T) {
	t.Parallel()

	store, application := openEvalApp(t)
	chun := lookupNamed(t, store, "Chun-Li")
	response := post(t, application, "/contacts/"+strconv.FormatInt(chun.ID, 10)+"/delete", url.Values{})
	assertStatus(t, response, http.StatusSeeOther)
	found, err := store.List(context.Background(), "Chun-Li", 1)
	if err != nil || len(found.Contacts) != 0 {
		t.Fatalf("Chun-Li still present: %#v %v", found.Contacts, err)
	}
}

func TestEvalOracleCountRyu(t *testing.T) {
	t.Parallel()

	store, application := openEvalApp(t)
	search := get(t, application, "/contacts?q=Ryu")
	assertStatus(t, search, http.StatusOK)
	assertBody(t, search, "Ryu", "Kenji")
	total, err := store.CountNamed(context.Background(), "Ryu")
	if err != nil || total != 3 {
		t.Fatalf("Ryu count = %d %v, want 3", total, err)
	}
}

func openEvalApp(t *testing.T) (*contact.Store, http.Handler) {
	t.Helper()
	store, application := openApp(t)
	if err := store.SeedEvalIfEmpty(context.Background()); err != nil {
		t.Fatalf("seed eval: %v", err)
	}
	return store, application
}

func lookupNamed(t *testing.T, store *contact.Store, query string) contact.Contact {
	t.Helper()
	found, err := store.List(context.Background(), query, 1)
	if err != nil || len(found.Contacts) == 0 {
		t.Fatalf("lookup %q: %#v %v", query, found.Contacts, err)
	}
	return found.Contacts[0]
}
