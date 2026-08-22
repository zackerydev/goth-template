package templates_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/zackerydev/goth-template/templates"
)

func TestHomeRendersApplicationShell(t *testing.T) {
	t.Parallel()

	var rendered bytes.Buffer
	if err := templates.Home().Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render home page: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(rendered.String()), "<!doctype html>") {
		t.Fatal("home page does not start with an HTML doctype")
	}

	document, err := goquery.NewDocumentFromReader(strings.NewReader(rendered.String()))
	if err != nil {
		t.Fatalf("parse home page: %v", err)
	}
	assertAttribute(t, document, "html", expectedAttribute{name: "lang", value: "en"})
	assertAttribute(t, document, `meta[name="viewport"]`, expectedAttribute{name: "content", value: "width=device-width, initial-scale=1"})
	assertAttribute(t, document, `link[rel="stylesheet"]`, expectedAttribute{name: "href", value: "/assets/css/app.css"})
	assertAttribute(t, document, `button[hx-get]`, expectedAttribute{name: "hx-get", value: "/greeting"})
	assertAttribute(t, document, `button[hx-target]`, expectedAttribute{name: "hx-target", value: "#greeting"})

	if title := strings.TrimSpace(document.Find("title").Text()); title != "GoTH Template" {
		t.Errorf("title = %q, want %q", title, "GoTH Template")
	}
	if got := document.Find("main#main-content h1").Length(); got != 1 {
		t.Errorf("main heading count = %d, want 1", got)
	}
	if got := document.Find(`script[src="/assets/js/htmx.min.js"]`).Length(); got != 1 {
		t.Errorf("htmx script count = %d, want 1", got)
	}
	if got := document.Find(`script[src="/assets/js/app.js"]`).Length(); got != 1 {
		t.Errorf("application script count = %d, want 1", got)
	}
}

func TestGreetingRendersFragment(t *testing.T) {
	t.Parallel()

	var rendered bytes.Buffer
	if err := templates.Greeting().Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render greeting: %v", err)
	}
	markup := rendered.String()
	if strings.Contains(markup, "<html") || !strings.Contains(markup, `id="greeting"`) {
		t.Fatalf("greeting is not a fragment: %s", markup)
	}
}

type expectedAttribute struct {
	name  string
	value string
}

func assertAttribute(t *testing.T, document *goquery.Document, selector string, attribute expectedAttribute) {
	t.Helper()
	selection := document.Find(selector)
	if selection.Length() != 1 {
		t.Fatalf("%s count = %d, want 1", selector, selection.Length())
	}
	if got := selection.AttrOr(attribute.name, ""); got != attribute.value {
		t.Errorf("%s %s = %q, want %q", selector, attribute.name, got, attribute.value)
	}
}
