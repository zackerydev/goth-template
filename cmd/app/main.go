package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/zackerydev/goth-template/internal/contact"
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
	return newApplicationWithTemplateDirectory(development, "templates", databasePath())
}

func newApplicationWithTemplateDirectory(development bool, directory, database string) (http.Handler, error) {
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
	store, err := contact.Open(database)
	if err != nil {
		return nil, err
	}
	if err := seedStore(store); err != nil {
		_ = store.Close()
		return nil, err
	}
	return server.New(func(mux *http.ServeMux) {
		handler.Register(mux, renderer, store)
	}), nil
}

func seedStore(store *contact.Store) error {
	ctx := context.Background()
	if os.Getenv("APP_SEED") == "eval" {
		return store.SeedEvalIfEmpty(ctx)
	}
	return store.SeedIfEmpty(ctx)
}

func databasePath() string {
	if value := os.Getenv("APP_DATABASE"); value != "" {
		return value
	}
	return filepath.Join("tmp", "contacts.db")
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
