package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/zackerydev/goth-template/internal/server"
)

func main() {
	port, err := applicationPort()
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           server.New(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
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
