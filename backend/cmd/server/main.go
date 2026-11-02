// Command server runs the Currency Watcher API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"currencywatcher/internal/api"
	"currencywatcher/internal/cache"
	"currencywatcher/internal/frankfurter"
	"currencywatcher/internal/rates"
)

const (
	defaultPort           = "8080"
	defaultAllowedOrigins = "http://localhost:5173"
	defaultFrankfurterURL = "https://api.frankfurter.dev/v2"

	cacheTTL        = time.Hour
	upstreamTimeout = 10 * time.Second
	shutdownTimeout = 10 * time.Second
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	client := frankfurter.NewClient(
		getenv("FRANKFURTER_BASE_URL", defaultFrankfurterURL),
		&http.Client{Timeout: upstreamTimeout},
	)
	service := rates.NewService(client, cache.New[rates.Table](cacheTTL, time.Now))
	origins := parseOrigins(getenv("ALLOWED_ORIGINS", defaultAllowedOrigins))

	server := &http.Server{
		Addr:              ":" + getenv("PORT", defaultPort),
		Handler:           api.NewHandler(service, logger, origins),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      upstreamTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", server.Addr, "allowedOrigins", origins)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseOrigins(raw string) []string {
	var origins []string
	for _, origin := range strings.Split(raw, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}
