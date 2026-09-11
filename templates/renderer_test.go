package templates_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/zackerydev/goth-template/templates"
)

func TestEmbeddedTemplatesRenderPagesAndPartials(t *testing.T) {
	t.Parallel()

	renderer, err := templates.New()
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}

	var page bytes.Buffer
	if err := renderer.Render(&page, "contacts", map[string]any{
		"Title": "Contacts",
		"Flash": "Created New Contact!",
		"Query": "ada",
		"Contacts": []map[string]any{
			{"ID": 1, "First": "Ada", "Last": "Lovelace", "Phone": "1", "Email": "ada@example.com"},
		},
		"HasMore":  true,
		"NextPage": 2,
	}); err != nil {
		t.Fatalf("render contacts: %v", err)
	}
	for _, want := range []string{
		"<!doctype html>",
		`<html lang="en">`,
		`id="main-content"`,
		`hx-boost:inherited="swap:outerSync select:#main-content target:#main-content"`,
		"contacts.app",
		"Ada",
		"Created New Contact!",
		`hx-get="/contacts"`,
		"Loading More…",
	} {
		if !strings.Contains(page.String(), want) {
			t.Errorf("contacts does not contain %q: %s", want, page.String())
		}
	}

	var form bytes.Buffer
	if err := renderer.Render(&form, "contacts-new", map[string]any{
		"Title":   "New Contact",
		"Flash":   "",
		"Contact": stubContact{},
	}); err != nil {
		t.Fatalf("render new contact: %v", err)
	}
	if !strings.Contains(form.String(), `action="/contacts/new"`) {
		t.Errorf("new contact = %q", form.String())
	}
}

func TestProductionDoesNotReadSourceAfterInitialization(t *testing.T) {
	t.Parallel()

	source := &countingFS{FS: fixtureFS("home")}
	renderer, err := templates.NewProductionFS(source)
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	source.opens.Store(0)

	var output bytes.Buffer
	if err := renderer.Render(&output, "home", map[string]string{"Title": "Home"}); err != nil {
		t.Fatalf("render home: %v", err)
	}
	if err := renderer.Render(&output, "greeting", nil); err != nil {
		t.Fatalf("render greeting: %v", err)
	}
	if got := source.opens.Load(); got != 0 {
		t.Errorf("source opens after initialization = %d, want 0", got)
	}
}

func TestProductionPagesHaveIndependentTemplateSets(t *testing.T) {
	t.Parallel()

	renderer, err := templates.NewProductionFS(fixtureFS("alpha", "beta"))
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	for _, test := range []struct{ name, want, omit string }{
		{name: "alpha", want: "alpha page", omit: "beta page"},
		{name: "beta", want: "beta page", omit: "alpha page"},
	} {
		var output bytes.Buffer
		if err := renderer.Render(&output, test.name, map[string]string{"Title": test.name}); err != nil {
			t.Fatalf("render %s: %v", test.name, err)
		}
		if !strings.Contains(output.String(), test.want) || strings.Contains(output.String(), test.omit) {
			t.Errorf("%s output = %q", test.name, output.String())
		}
	}
}

func TestRendererEscapesValuesAndReportsFailures(t *testing.T) {
	t.Parallel()

	renderer, err := templates.NewProductionFS(fixtureFS("value"))
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}

	var escaped bytes.Buffer
	if err := renderer.Render(&escaped, "value", map[string]string{
		"Title": "<title>",
		"Value": `<script>alert("x")</script>`,
	}); err != nil {
		t.Fatalf("render escaped value: %v", err)
	}
	if strings.Contains(escaped.String(), `<script>alert`) || !strings.Contains(escaped.String(), `&lt;script&gt;`) {
		t.Errorf("value was not escaped: %s", escaped.String())
	}
	if err := renderer.Render(&bytes.Buffer{}, "value", struct{}{}); err == nil {
		t.Fatal("render with missing field succeeded")
	}
	if err := renderer.Render(failingWriter{}, "value", map[string]string{"Title": "title", "Value": "value"}); err == nil {
		t.Fatal("render with failing writer succeeded")
	}
	if err := renderer.Render(&bytes.Buffer{}, "unknown", nil); err == nil {
		t.Fatal("render with unknown template succeeded")
	}
	for _, name := range []string{"", "../home", "pages/home", `home\page`} {
		if err := renderer.Render(&bytes.Buffer{}, name, nil); err == nil {
			t.Errorf("render with invalid name %q succeeded", name)
		}
	}
}

func TestDevelopmentReloadsAndRecovers(t *testing.T) {
	t.Parallel()

	source := fixtureFS("home")
	renderer, err := templates.NewDevelopmentFS(source)
	if err != nil {
		t.Fatalf("create development renderer: %v", err)
	}

	var initial bytes.Buffer
	if err := renderer.Render(&initial, "home", map[string]string{"Title": "Home"}); err != nil {
		t.Fatalf("render initial home: %v", err)
	}
	if !strings.Contains(initial.String(), "home page") {
		t.Fatalf("initial output = %q", initial.String())
	}
	if err := renderer.Render(&bytes.Buffer{}, "greeting", nil); err != nil {
		t.Fatalf("render development partial: %v", err)
	}

	source["pages/home.html"].Data = []byte(`{{ define "home" }}`)
	if err := renderer.Render(&bytes.Buffer{}, "home", nil); err == nil {
		t.Fatal("malformed development template succeeded")
	}

	source["pages/home.html"].Data = []byte(`{{ define "home" }}{{ template "layout" . }}{{ end }}{{ define "content" }}home two{{ end }}`)
	source["layouts/layout.html"].Data = []byte(`{{ define "layout" }}layout two {{ template "content" . }}{{ end }}`)
	var updated bytes.Buffer
	if err := renderer.Render(&updated, "home", nil); err != nil {
		t.Fatalf("render updated home: %v", err)
	}
	if !strings.Contains(updated.String(), "layout two") || !strings.Contains(updated.String(), "home two") {
		t.Errorf("updated output = %q", updated.String())
	}

	if err := renderer.Render(&bytes.Buffer{}, "missing", nil); err == nil {
		t.Error("render missing development template succeeded")
	}
}

func TestDevelopmentLoadsFromDisk(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	for name, file := range fixtureFS("home") {
		filename := filepath.Join(directory, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatalf("create template directory: %v", err)
		}
		if err := os.WriteFile(filename, file.Data, 0o600); err != nil {
			t.Fatalf("write template %q: %v", name, err)
		}
	}

	renderer, err := templates.NewDevelopment(directory)
	if err != nil {
		t.Fatalf("create development renderer: %v", err)
	}
	var output bytes.Buffer
	if err := renderer.Render(&output, "home", map[string]string{"Title": "Disk"}); err != nil {
		t.Fatalf("render disk templates: %v", err)
	}
	if !strings.Contains(output.String(), "home page") {
		t.Errorf("disk output = %q", output.String())
	}
}

func TestRendererInitializationErrors(t *testing.T) {
	t.Parallel()

	if _, err := templates.NewProductionFS(nil); err == nil {
		t.Error("nil production source succeeded")
	}
	if _, err := templates.NewDevelopmentFS(nil); err == nil {
		t.Error("nil development source succeeded")
	}
	if _, err := templates.NewDevelopment(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("missing development directory succeeded")
	}
	file := filepath.Join(t.TempDir(), "templates.html")
	if err := os.WriteFile(file, []byte("templates"), 0o600); err != nil {
		t.Fatalf("write temporary file: %v", err)
	}
	if _, err := templates.NewDevelopment(file); err == nil {
		t.Error("file development path succeeded")
	}
}

func TestProductionTemplateErrors(t *testing.T) {
	t.Parallel()

	malformed := fixtureFS("home")
	malformed["partials/greeting.html"].Data = []byte(`{{ define "greeting" }}`)
	if _, err := templates.NewProductionFS(malformed); err == nil {
		t.Error("malformed production template succeeded")
	}
	malformedPage := fixtureFS("home")
	malformedPage["pages/home.html"].Data = []byte(`{{ define "home" }}`)
	if _, err := templates.NewProductionFS(malformedPage); err == nil {
		t.Error("malformed production page succeeded")
	}
	missingLayouts := fixtureFS("home")
	delete(missingLayouts, "layouts/layout.html")
	if _, err := templates.NewProductionFS(missingLayouts); err == nil {
		t.Error("production source without layouts succeeded")
	}
	missingPartials := fixtureFS("home")
	delete(missingPartials, "partials/document-title.html")
	delete(missingPartials, "partials/greeting.html")
	if _, err := templates.NewProductionFS(missingPartials); err == nil {
		t.Error("production source without partials succeeded")
	}
	missingPages := fixtureFS("home")
	delete(missingPages, "pages/home.html")
	if _, err := templates.NewProductionFS(missingPages); err == nil {
		t.Error("production source without pages succeeded")
	}
	duplicate := fixtureFS("home")
	duplicate["pages/greeting.html"] = &fstest.MapFile{Data: []byte(`{{ define "greeting" }}page{{ end }}`)}
	if _, err := templates.NewProductionFS(duplicate); err == nil {
		t.Error("production source with duplicate names succeeded")
	}
}

func TestDevelopmentTemplateErrors(t *testing.T) {
	t.Parallel()

	developmentWithoutLayout := fixtureFS("home")
	delete(developmentWithoutLayout, "layouts/layout.html")
	developmentRenderer, err := templates.NewDevelopmentFS(developmentWithoutLayout)
	if err != nil {
		t.Fatalf("create development renderer without layout: %v", err)
	}
	if err := developmentRenderer.Render(&bytes.Buffer{}, "home", nil); err == nil {
		t.Error("development source without layout succeeded")
	}
	developmentWithoutPartials := fixtureFS("home")
	delete(developmentWithoutPartials, "partials/document-title.html")
	delete(developmentWithoutPartials, "partials/greeting.html")
	developmentRenderer, err = templates.NewDevelopmentFS(developmentWithoutPartials)
	if err != nil {
		t.Fatalf("create development renderer without partials: %v", err)
	}
	if err := developmentRenderer.Render(&bytes.Buffer{}, "home", nil); err == nil {
		t.Error("development source without partials succeeded")
	}

	sentinel := errors.New("source unavailable")
	errorRenderer, err := templates.NewDevelopmentFS(errorFS{err: sentinel})
	if err != nil {
		t.Fatalf("create development renderer: %v", err)
	}
	if err := errorRenderer.Render(&bytes.Buffer{}, "home", nil); !errors.Is(err, sentinel) {
		t.Errorf("source error = %v, want %v", err, sentinel)
	}
	partialErrorRenderer, err := templates.NewDevelopmentFS(partialErrorFS{err: sentinel})
	if err != nil {
		t.Fatalf("create development renderer with partial error source: %v", err)
	}
	if err := partialErrorRenderer.Render(&bytes.Buffer{}, "missing", nil); !errors.Is(err, sentinel) {
		t.Errorf("partial source error = %v, want %v", err, sentinel)
	}

	var nilRenderer *templates.Renderer
	if err := nilRenderer.Render(&bytes.Buffer{}, "home", nil); err == nil {
		t.Error("nil renderer succeeded")
	}
}

func fixtureFS(pages ...string) fstest.MapFS {
	filesystem := fstest.MapFS{
		"layouts/layout.html":          &fstest.MapFile{Data: []byte(`{{ define "layout" }}<html>{{ template "document-title" . }}{{ template "content" . }}</html>{{ end }}`)},
		"layouts/unused/extra.html":    &fstest.MapFile{Data: []byte("ignored")},
		"layouts/notes.txt":            &fstest.MapFile{Data: []byte("ignored")},
		"partials/document-title.html": &fstest.MapFile{Data: []byte(`{{ define "document-title" }}<title>{{ .Title }}</title>{{ end }}`)},
		"partials/greeting.html":       &fstest.MapFile{Data: []byte(`{{ define "greeting" }}initial greeting{{ end }}`)},
	}
	for _, page := range pages {
		content := fmt.Sprintf(`{{ define %q }}{{ template "layout" . }}{{ end }}{{ define "content" }}%s{{ end }}`, page, page+" page")
		if page == "home" {
			content = `{{ define "home" }}{{ template "layout" . }}{{ end }}{{ define "content" }}home page {{ template "greeting" . }}{{ end }}`
		}
		if page == "value" {
			content = `{{ define "value" }}{{ template "layout" . }}{{ end }}{{ define "content" }}{{ .Value }}{{ end }}`
		}
		filesystem["pages/"+page+".html"] = &fstest.MapFile{Data: []byte(content)}
	}
	return filesystem
}

type stubContact struct {
	ID    int64
	First string
	Last  string
	Phone string
	Email string
}

func (stubContact) Error(string) string {
	return ""
}

type countingFS struct {
	fs.FS
	opens atomic.Int64
}

func (filesystem *countingFS) Open(name string) (fs.File, error) {
	filesystem.opens.Add(1)
	return filesystem.FS.Open(name)
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("writer failed")
}

type errorFS struct {
	err error
}

func (filesystem errorFS) Open(string) (fs.File, error) {
	return nil, filesystem.err
}

type partialErrorFS struct {
	err error
}

func (filesystem partialErrorFS) Open(name string) (fs.File, error) {
	if name == "pages/missing.html" {
		return nil, fs.ErrNotExist
	}
	return nil, filesystem.err
}
