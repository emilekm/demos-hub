package config

import (
	"os"

	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
)

type Config struct {
	ListenAddr string    `yaml:"listenAddr"`
	SpaceUUID  uuid.UUID `yaml:"spaceUUID"`

	UploadDir string `yaml:"uploadDir"`
	UploadURL string `yaml:"uploadURL"`
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
