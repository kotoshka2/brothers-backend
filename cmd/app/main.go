package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"brothers-backend/internal/application"
	"brothers-backend/internal/infrastructure/sqlite"
	"brothers-backend/internal/presentation"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")
	jwtSecret := os.Getenv("JWT_SECRET")
	if username == "" || len(password) < 12 || len(jwtSecret) < 32 {
		return errors.New("ADMIN_USERNAME, ADMIN_PASSWORD (at least 12 characters), and JWT_SECRET (at least 32 characters) are required")
	}

	passwordHash, err := application.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	databasePath := os.Getenv("DATABASE_PATH")
	if databasePath == "" {
		databasePath = "data/brothers.db"
	}
	startupContext, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	repository, err := sqlite.Open(startupContext, databasePath)
	cancelStartup()
	if err != nil {
		return err
	}
	defer func() {
		if err := repository.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	auth := application.NewAuthService(username, passwordHash, []byte(jwtSecret), repository)
	content := application.NewContentService(repository)
	api := presentation.NewAPI(content, auth, strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ","))

	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = "0.0.0.0:8080"
	}
	server := &http.Server{
		Addr: address, Handler: api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("HTTP server listening on %s", address)
		serverErrors <- server.ListenAndServe()
	}()
	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-signalContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	}
}
