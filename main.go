package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/emilekm/demos-hub/internal/config"
	"github.com/emilekm/demos-hub/internal/rmod"
	"github.com/google/uuid"
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

	serversAPI := &ServersAPI{
		uploadDir: cfg.UploadDir,
		uploadURL: cfg.UploadURL,
		space:     cfg.SpaceUUID,
	}

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

type ServersAPI struct {
	space     uuid.UUID
	uploadDir string
	uploadURL string
}

func (s *ServersAPI) Servers(w http.ResponseWriter, r *http.Request) {
	serversDirs, err := os.ReadDir(s.uploadDir)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	servers := make([]string, 0)
	for _, server := range serversDirs {
		if !server.IsDir() {
			continue
		}

		servers = append(servers, server.Name())
	}

	payload := struct {
		Servers []string `json:"servers"`
	}{
		Servers: servers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func (s *ServersAPI) ServerFiles(w http.ResponseWriter, r *http.Request) {
	server := r.PathValue("server")
	if server == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	serverDir := filepath.Join(s.uploadDir, server)
	files, err := os.ReadDir(serverDir)
	if err != nil {
		http.Error(w, "Server doesn't exist", http.StatusBadRequest)
		return
	}

	serverFiles := make([]string, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		serverFiles = append(serverFiles, path.Join(s.uploadURL, server, file.Name()))
	}

	payload := struct {
		Files []string `json:"files"`
	}{
		Files: serverFiles,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(payload)
	if err != nil {
		slog.Error("failed to encode response", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *ServersAPI) UploadFile(w http.ResponseWriter, r *http.Request) {
	ip := r.Header.Get("X-PRHub-IP")
	port := r.Header.Get("X-PRHub-Port")
	license := r.Header.Get("X-PRHub-License")
	if ip == "" || port == "" || license == "" {
		http.Error(w, "Missing headers", http.StatusUnauthorized)
		return
	}

	valid, err := rmod.ValidateLicense(ip, port, license)
	if err != nil || !valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	serverID := s.serverID(license)

	attachment, header, err := r.FormFile("prdemo")
	defer attachment.Close()
	if err != nil {
		slog.Error("failed to read form file", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = os.MkdirAll(filepath.Join(s.uploadDir, serverID), 0755)
	if err != nil {
		slog.Error("failed to create server dir", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	file, err := os.Create(filepath.Join(s.uploadDir, serverID, header.Filename))
	defer file.Close()
	if err != nil {
		slog.Error("failed to create file", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(file, attachment)
	if err != nil {
		slog.Error("failed to write file", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	payload := struct {
		Server struct {
			ID string `json:"id"`
		} `json:"server"`
		File struct {
			URL string `json:"url"`
		} `json:"file"`
	}{
		Server: struct {
			ID string `json:"id"`
		}{
			ID: serverID,
		},
		File: struct {
			URL string `json:"url"`
		}{
			URL: path.Join(s.uploadURL, serverID, header.Filename),
		},
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(payload)
	if err != nil {
		slog.Error("failed to encode response", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *ServersAPI) serverID(license string) string {
	return uuid.NewMD5(s.space, []byte(license)).String()
}
