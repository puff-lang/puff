package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrConfigNotFound = errors.New("puff.toml not found")

func FindConfigPath(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve start directory: %w", err)
	}

	for {
		path := filepath.Join(dir, ConfigFileName)

		info, err := os.Stat(path)
		if err == nil {
			if info.IsDir() {
				return "", fmt.Errorf("config path is a directory: %s", path)
			}

			return path, nil
		}

		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("check config path: %w", err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrConfigNotFound
		}

		dir = parent
	}
}

func LoadNearestConfig(startDir string) (*Config, string, error) {
	path, err := FindConfigPath(startDir)
	if err != nil {
		return nil, "", err
	}

	config, err := LoadConfig(path)
	if err != nil {
		return nil, "", err
	}

	return config, filepath.Dir(path), nil
}
