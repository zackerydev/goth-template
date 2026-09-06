package handler

import (
	"bytes"
	"net/http"

	"github.com/zackerydev/goth-template/templates"
)

const homeTitle = "GoTH Template"

// Home returns the home page handler.
func Home(renderer *templates.Renderer) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		var rendered bytes.Buffer
		if err := renderer.Render(&rendered, "home", templates.PageData{Title: homeTitle}); err != nil {
			http.Error(writer, "template rendering failed", http.StatusInternalServerError)
			return
		}
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
