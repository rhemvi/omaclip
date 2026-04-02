// Package config handles loading and saving the Clipmaster config file.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DefaultPath returns the default path for the config file.
func DefaultPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config/clipmaster/config.json")
}

// Config holds persistent application configuration.
type Config struct {
	Passphrase string `json:"passphrase"`
}

// Load reads and unmarshals the config file at the given path.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Save marshals cfg and writes it to path, creating parent directories as needed.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
