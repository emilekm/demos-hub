package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"time"

	v1 "github.com/emilekm/demos-hub/internal/api/v1"
	"github.com/emilekm/demos-hub/internal/config"
)

var configFile string
var serverName = "hub"

func main() {
	flag.StringVar(&configFile, "config", "config.yaml", "config file")
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.New(configFile)
	if err != nil {
		return err
	}

	serversAPI := v1.NewServers(cfg.SpaceUUID, cfg.UploadDir, cfg.UploadURL)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /servers", serversAPI.Servers)
	mux.HandleFunc("POST /upload", WithLogging(serversAPI.UploadFile))
	mux.HandleFunc("GET /servers/{server}", serversAPI.ServerFiles)

	log.Printf("Listening on %s", cfg.ListenAddr)

	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		return err
	}

	return nil
}

func WithLogging(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()

		uri := r.RequestURI
		method := r.Method
		h.ServeHTTP(rw, r) // serve the original request

		duration := time.Since(start)

		// log request details
		slog.Info("",
			"uri", uri,
			"method", method,
			"duration", duration,
		)
	}
	return logFn
}
