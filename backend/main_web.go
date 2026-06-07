//go:build web

package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//go:embed all:frontend/dist
var webAssets embed.FS

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		if err := checkWebHealth(envOrDefault("HEALTHCHECK_URL", "http://127.0.0.1:8080/healthz")); err != nil {
			log.Fatal(err)
		}
		return
	}

	address := envOrDefault("WEB_ADDR", ":8080")
	dataDir := envOrDefault("DATA_DIR", "/data")
	downloadDir := envOrDefault("DOWNLOAD_DIR", "/downloads")

	app := NewAppWithPaths(dataDir, downloadDir)
	app.start()
	defer app.stop()

	assets, err := fs.Sub(webAssets, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              address,
		Handler:           newWebHandler(app, assets, downloadDir),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("web server listening on %s", address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("web server shutdown: %v", err)
	}
}

func checkWebHealth(url string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck returned HTTP %d", response.StatusCode)
	}
	return nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
