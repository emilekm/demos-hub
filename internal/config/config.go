package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	UploadDir  string `yaml:"uploadDir"`
	ListenAddr string `yaml:"listenAddr"`
}

func New(filename string) (*Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
