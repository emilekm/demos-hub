package v1

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/emilekm/demos-hub/internal/rmod"
	"github.com/google/uuid"
)

type Servers struct {
	space     uuid.UUID
	uploadDir string
	uploadURL string
}

func NewServers(space uuid.UUID, uploadDir, uploadURL string) *Servers {
	return &Servers{
		space:     space,
		uploadDir: uploadDir,
		uploadURL: uploadURL,
	}
}

type server struct {
	ID string `json:"id"`
}

func (s *Servers) Servers(w http.ResponseWriter, r *http.Request) {
	dirs, err := os.ReadDir(s.uploadDir)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	servers := make([]server, 0)
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		servers = append(servers, server{
			ID: dir.Name(),
		})
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

	serverFiles := make([]serverFile, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		serverFiles = append(serverFiles, serverFile{
			Name: file.Name(),
			URL:  path.Join(s.uploadURL, server, file.Name()),
		})
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
