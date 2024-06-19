package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/prbf2-tools/demos-hub/internal/config"
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
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /servers", serversAPI.Servers)
	mux.HandleFunc("GET /servers/{server}", serversAPI.ServerFiles)
	mux.HandleFunc("POST /servers/{server}", serversAPI.UploadFile)

	log.Printf("Listening on %s", cfg.ListenAddr)

	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		return err
	}

	return nil
}

type ServersAPI struct {
	uploadDir string
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

		serverFiles = append(serverFiles, file.Name())
	}

	payload := struct {
		Files []string `json:"files"`
	}{
		Files: serverFiles,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func (s *ServersAPI) UploadFile(w http.ResponseWriter, r *http.Request) {
	server := r.PathValue("server")
	if server == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	attachment, header, err := r.FormFile("prdemo")
	defer attachment.Close()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	file, err := os.Create(filepath.Join(s.uploadDir, server, header.Filename))
	defer file.Close()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(file, attachment)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
