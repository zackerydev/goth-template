package handler

import (
	"bytes"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/zackerydev/goth-template/internal/contact"
	"github.com/zackerydev/goth-template/templates"
)

const flashCookie = "flash"

type pageWrite struct {
	renderer *templates.Renderer
	name     string
	data     any
	status   int
}

func writePage(writer http.ResponseWriter, page pageWrite) {
	if page.status == 0 {
		page.status = http.StatusOK
	}
	var rendered bytes.Buffer
	if err := page.renderer.Render(&rendered, page.name, page.data); err != nil {
		http.Error(writer, "template rendering failed", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(page.status)
	_, _ = writer.Write(rendered.Bytes())
}

func seeOther(writer http.ResponseWriter, request *http.Request, location string) {
	if request.Header.Get("HX-Request") == "true" && request.Header.Get("HX-Boosted") != "true" {
		writer.Header().Set("HX-Redirect", location)
		writer.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(writer, request, location, http.StatusSeeOther)
}

func setFlash(writer http.ResponseWriter, message string) {
	http.SetCookie(writer, newFlashCookie(url.QueryEscape(message), 60))
}

func takeFlash(writer http.ResponseWriter, request *http.Request) string {
	cookie, err := request.Cookie(flashCookie)
	if err != nil {
		return ""
	}
	http.SetCookie(writer, newFlashCookie("", -1))
	message, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return ""
	}
	return message
}

func newFlashCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // G124: served over local HTTP; Secure would drop the flash cookie.
		Name:     flashCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func parsePage(value string) int {
	page, err := strconv.Atoi(value)
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func formContact(request *http.Request) contact.Contact {
	return contact.Contact{
		First: strings.TrimSpace(request.FormValue("first_name")),
		Last:  strings.TrimSpace(request.FormValue("last_name")),
		Phone: strings.TrimSpace(request.FormValue("phone")),
		Email: strings.TrimSpace(request.FormValue("email")),
	}
}
