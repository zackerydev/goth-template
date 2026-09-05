package templates

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

// PageData contains values shared by a page and its document title.
type PageData struct {
	Title string
}

// Renderer executes the application's HTML templates.
type Renderer struct {
	production  map[string]*template.Template
	source      fs.FS
	development bool
}

// New loads and validates the embedded production templates.
func New() (*Renderer, error) {
	return NewProductionFS(embeddedTemplates)
}

// NewProductionFS loads and validates templates from source.
func NewProductionFS(source fs.FS) (*Renderer, error) {
	if source == nil {
		return nil, errors.New("template source is nil")
	}

	loaded, err := loadProduction(source)
	if err != nil {
		return nil, err
	}
	return &Renderer{production: loaded}, nil
}

// NewDevelopment loads templates from a directory and reparses them per render.
func NewDevelopment(directory string) (*Renderer, error) {
	info, err := os.Stat(directory)
	if err != nil {
		return nil, fmt.Errorf("stat template directory %q: %w", directory, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("template path %q is not a directory", directory)
	}
	return NewDevelopmentFS(os.DirFS(directory))
}

// NewDevelopmentFS creates a renderer that reparses templates from source per render.
func NewDevelopmentFS(source fs.FS) (*Renderer, error) {
	if source == nil {
		return nil, errors.New("template source is nil")
	}
	return &Renderer{source: source, development: true}, nil
}

// Render executes the named page or standalone partial into writer.
func (r *Renderer) Render(writer io.Writer, name string, data any) error {
	if r == nil {
		return errors.New("template renderer is nil")
	}

	parsed, err := r.template(name)
	if err != nil {
		return err
	}
	if err := parsed.ExecuteTemplate(writer, name, data); err != nil {
		return fmt.Errorf("render template %q: %w", name, err)
	}
	return nil
}

func (r *Renderer) template(name string) (*template.Template, error) {
	if !validTemplateName(name) {
		return nil, fmt.Errorf("invalid template name %q", name)
	}
	if r.development {
		return loadDevelopment(r.source, name)
	}
	parsed, ok := r.production[name]
	if !ok {
		return nil, fmt.Errorf("template %q not found", name)
	}
	return parsed, nil
}

func loadProduction(source fs.FS) (map[string]*template.Template, error) {
	layouts, err := templateFiles(source, "layouts")
	if err != nil {
		return nil, err
	}
	partials, err := templateFiles(source, "partials")
	if err != nil {
		return nil, err
	}
	pages, err := templateFiles(source, "pages")
	if err != nil {
		return nil, err
	}

	loaded := make(map[string]*template.Template, len(pages)+len(partials))
	for _, partialPath := range partials {
		name := templateFileName(partialPath, "partials")
		parsed, err := parseFiles(source, name, partialPath)
		if err != nil {
			return nil, err
		}
		loaded[name] = parsed
	}
	for _, pagePath := range pages {
		name := templateFileName(pagePath, "pages")
		if _, exists := loaded[name]; exists {
			return nil, fmt.Errorf("duplicate template name %q", name)
		}
		files := append(append([]string{}, layouts...), partials...)
		files = append(files, pagePath)
		parsed, err := parseFiles(source, name, files...)
		if err != nil {
			return nil, err
		}
		loaded[name] = parsed
	}
	return loaded, nil
}

func loadDevelopment(source fs.FS, name string) (*template.Template, error) {
	pagePath := path.Join("pages", name+".html")
	pageExists, err := fileExists(source, pagePath)
	if err != nil {
		return nil, err
	}
	if pageExists {
		layouts, err := templateFiles(source, "layouts")
		if err != nil {
			return nil, err
		}
		partials, err := templateFiles(source, "partials")
		if err != nil {
			return nil, err
		}
		files := append(append([]string{}, layouts...), partials...)
		files = append(files, pagePath)
		return parseFiles(source, name, files...)
	}

	partialPath := path.Join("partials", name+".html")
	partialExists, err := fileExists(source, partialPath)
	if err != nil {
		return nil, err
	}
	if partialExists {
		return parseFiles(source, name, partialPath)
	}
	return nil, fmt.Errorf("template %q not found", name)
}

func parseFiles(source fs.FS, name string, files ...string) (*template.Template, error) {
	parsed, err := template.New(name).ParseFS(source, files...)
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", name, err)
	}
	return parsed, nil
}

func templateFiles(source fs.FS, directory string) ([]string, error) {
	entries, err := fs.ReadDir(source, directory)
	if err != nil {
		return nil, fmt.Errorf("find %s templates: %w", directory, err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".html" {
			continue
		}
		files = append(files, path.Join(directory, entry.Name()))
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no templates found in %s", directory)
	}
	return files, nil
}

func fileExists(source fs.FS, name string) (bool, error) {
	_, err := fs.Stat(source, name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("stat template %q: %w", name, err)
}

func templateFileName(filename, directory string) string {
	return strings.TrimSuffix(strings.TrimPrefix(filename, directory+"/"), ".html")
}

func validTemplateName(name string) bool {
	return name != "" && path.Base(name) == name && !strings.Contains(name, `\`)
}

//go:embed layouts/*.html pages/*.html partials/*.html
var embeddedTemplates embed.FS
