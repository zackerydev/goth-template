package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/zackerydev/goth-template/internal/handler"
	"github.com/zackerydev/goth-template/internal/server"
	"github.com/zackerydev/goth-template/templates"
)

func main() {
	development := flag.Bool("dev", false, "reload HTML templates from disk")
	flag.Parse()

	application, err := newApplication(*development)
	if err != nil {
		log.Fatal(err)
	}
	port, err := applicationPort()
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           application,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func newApplication(development bool) (http.Handler, error) {
	return newApplicationWithTemplateDirectory(development, "templates")
}

func newApplicationWithTemplateDirectory(development bool, directory string) (http.Handler, error) {
	var (
		renderer *templates.Renderer
		err      error
	)
	if development {
		renderer, err = templates.NewDevelopment(directory)
	} else {
		renderer, err = templates.New()
	}
	if err != nil {
		return nil, err
	}
	return server.New(http.RedirectHandler("/lab", http.StatusSeeOther), handler.Greeting(renderer), handler.NewLab(renderer)), nil
}

func applicationPort() (int, error) {
	value := os.Getenv("APP_PORT")
	if value == "" {
		value = os.Getenv("PORT")
	}
	if value == "" {
		return 8888, nil
	}

	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, errors.New("application port must be an integer between 1 and 65535")
	}
	return port, nil
}
