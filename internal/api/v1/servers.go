package v1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"path"

	"github.com/emilekm/demos-hub/internal/rmod"
	"github.com/emilekm/demos-hub/internal/storage"
	"github.com/google/uuid"
)

type Servers struct {
	storage   *storage.Storage
	space     uuid.UUID
	uploadURL string
}

func NewServers(storage *storage.Storage, space uuid.UUID, uploadURL string) *Servers {
	return &Servers{
		space:     space,
		storage:   storage,
		uploadURL: uploadURL,
	}
}

func (s *Servers) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /upload", s.UploadFile)
	mux.HandleFunc("GET /servers", s.Servers)
	mux.HandleFunc("GET /servers/{serverID}", s.ServerFiles)

	return mux
}

type server struct {
	ID string `json:"id"`
}

func (s *Servers) Servers(w http.ResponseWriter, r *http.Request) {
	serversList, err := s.storage.ListServers()
	if err != nil {
		slog.Error("failed to list servers", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	servers := make([]server, len(serversList))
	for i, serverID := range serversList {
		servers[i] = server{
			ID: serverID,
		}
	}

	payload := struct {
		Servers []server `json:"servers"`
	}{
		Servers: servers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

type serverFile struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

func (s *Servers) ServerFiles(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("serverID")
	if serverID == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	files, err := s.storage.ListServerFiles(serverID)
	if err != nil {
		http.Error(w, "Server doesn't exist", http.StatusBadRequest)
		return
	}

	serverFiles := make([]serverFile, len(files))
	for i, file := range files {
		serverFiles[i] = serverFile{
			Name: file,
			URL:  path.Join(s.uploadURL, serverID, file),
		}
	}

	payload := struct {
		Files []serverFile `json:"files"`
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

func (s *Servers) UploadFile(w http.ResponseWriter, r *http.Request) {
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

	err = s.storage.SaveFile(serverID, header.Filename, attachment)
	if err != nil {
		slog.Error("failed to save file", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	payload := struct {
		Server server     `json:"server"`
		File   serverFile `json:"file"`
	}{
		Server: server{
			ID: serverID,
		},
		File: serverFile{
			Name: header.Filename,
			URL:  path.Join(s.uploadURL, serverID, header.Filename),
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

func (s *Servers) serverID(license string) string {
	return uuid.NewMD5(s.space, []byte(license)).String()
}
