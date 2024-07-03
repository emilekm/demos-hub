package storage

import (
	"io"
	"os"
	"path/filepath"
)

type Storage struct {
	uploadDir string
}

func NewStorage(uploadDir string) *Storage {
	return &Storage{
		uploadDir: uploadDir,
	}
}

func (s *Storage) SaveFile(serverID, filename string, file io.Reader) error {
	err := os.MkdirAll(filepath.Join(s.uploadDir, serverID), 0755)
	if err != nil {
		return err
	}

	osFile, err := os.Create(filepath.Join(s.uploadDir, serverID, filename))
	if err != nil {
		return err
	}
	defer osFile.Close()

	_, err = io.Copy(osFile, file)
	return err
}

func (s *Storage) ListServers() ([]string, error) {
	dirs, err := os.ReadDir(s.uploadDir)
	if err != nil {
		return nil, err
	}

	var servers []string
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		servers = append(servers, dir.Name())
	}

	return servers, nil
}

func (s *Storage) ListServerFiles(serverID string) ([]string, error) {
	serverDir := filepath.Join(s.uploadDir, serverID)
	files, err := os.ReadDir(serverDir)
	if err != nil {
		return nil, err
	}

	var serverFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		serverFiles = append(serverFiles, file.Name())
	}

	return serverFiles, nil
}
