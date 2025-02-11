package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
)

const DefaultTrashDir = "~/.myTrash"

type Config struct {
	TrashDir string `json:"trashDir"`
}

func NewConfig() (*Config, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, errors.Wrap(err, "load config")
	}

	return cfg, nil
}

func getConfigPathCandidate() []string {
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	homeDir := Home()

	paths := make([]string, 0)
	if xdgConfigHome != "" {
		paths = append(paths, filepath.Join(xdgConfigHome, "go-to-trash", "config.json"))
	}
	paths = append(paths, filepath.Join(homeDir, ".config", "go-to-trash", "config.json"))
	paths = append(paths, filepath.Join(homeDir, ".go-to-trash.json"))

	return paths
}

func loadConfig() (*Config, error) {
	for _, path := range getConfigPathCandidate() {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		defer file.Close()

		var cfg Config
		if err := json.NewDecoder(file).Decode(&cfg); err != nil {
			return nil, errors.Wrap(err, "decode config.json")
		}

		normalizedTrashDir, err := NormalizePath(cfg.TrashDir)
		if err != nil {
			return nil, errors.Wrap(err, "normalize trashDir")
		}

		if _, err := os.Stat(normalizedTrashDir); err != nil {
			log.Printf("%v is not exist. Create it.", normalizedTrashDir)
			return nil, errors.Wrap(err, "os.stat trashDir")
		}
		cfg.TrashDir = normalizedTrashDir
		return &cfg, nil
	}

	// no config file
	dir, _ := NormalizePath(DefaultTrashDir)
	return &Config{
		TrashDir: dir,
	}, nil
}
