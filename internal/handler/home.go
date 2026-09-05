package handler

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/zackerydev/goth-template/templates"
)

const homeTitle = "GoTH Template"

// Home returns the home page handler.
func Home(renderer *templates.Renderer) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		data := templates.PageData{Title: homeTitle}
		boosted := isTrue(request.Header.Get("HX-Boosted")) && !isTrue(request.Header.Get("HX-History-Restore-Request"))
		var rendered bytes.Buffer
		var err error
		if boosted {
			err = renderer.RenderDefinitions(&rendered, "home", []string{"document-title", "content"}, data)
		} else {
			err = renderer.Render(&rendered, "home", data)
		}
		if err != nil {
			http.Error(writer, "template rendering failed", http.StatusInternalServerError)
			return
		}

		addVary(writer.Header(), "HX-Boosted")
		addVary(writer.Header(), "HX-History-Restore-Request")
		writeHTML(writer, rendered.Bytes())
	})
}

// Greeting returns the example HTMX fragment handler.
func Greeting(renderer *templates.Renderer) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		var rendered bytes.Buffer
		if err := renderer.Render(&rendered, "greeting", nil); err != nil {
			http.Error(writer, "template rendering failed", http.StatusInternalServerError)
			return
		}
		writeHTML(writer, rendered.Bytes())
	})
}

func writeHTML(writer http.ResponseWriter, content []byte) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write(content); err != nil {
		return
	}
}

func addVary(header http.Header, value string) {
	for _, existing := range header.Values("Vary") {
		for item := range strings.SplitSeq(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(item), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}

func isTrue(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}
