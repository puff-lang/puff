package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, dir string, content string) string {
	t.Helper()

	path := filepath.Join(dir, ConfigFileName)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	return path
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()

	path := writeConfig(t, dir, `
	[pack]
	id = "example"

	[minecraft]
	versions = ">=1.21 <=1.21.6"
	target = "1.21.6"
	`)

	config, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.Pack.ID != "example" {
		t.Fatalf("expected pack id %q, got %q", "example", config.Pack.ID)
	}

	if config.Pack.Name != "example" {
		t.Fatalf("expected default pack name %q, got %q", "example", config.Pack.Name)
	}

	if config.Build.Source != DefaultSourceDir {
		t.Fatalf("expected default source %q, got %q", DefaultSourceDir, config.Build.Source)
	}

	if config.Build.Output != DefaultOutputDir {
		t.Fatalf("expected default output %q, got %q", DefaultOutputDir, config.Build.Output)
	}
}

func TestLoadConfigWithBuildSection(t *testing.T) {
	dir := t.TempDir()

	path := writeConfig(t, dir, `
	[pack]
	id = "example"
	name = "Example Pack"

	[minecraft]
	versions = ">=1.21 <=1.21.6"
	target = "1.21.6"
	pack_format = 48

	[build]
	source = "packs"
	output = "build"
	`)

	config, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.Pack.Name != "Example Pack" {
		t.Fatalf("expected pack name %q, got %q", "Example Pack", config.Pack.Name)
	}

	if config.Build.Source != "packs" {
		t.Fatalf("expected source %q, got %q", "packs", config.Build.Source)
	}

	if config.Build.Output != "build" {
		t.Fatalf("expected output %q, got %q", "build", config.Build.Output)
	}

	if config.Minecraft.PackFormat != 48 {
		t.Fatalf("expected pack format %d, got %d", 48, config.Minecraft.PackFormat)
	}
}

func TestLoadConfigRequiresPackID(t *testing.T) {
	dir := t.TempDir()

	path := writeConfig(t, dir, `
	[pack]
	name = "Example"

	[minecraft]
	versions = ">=1.21 <=1.21.6"
	`)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadConfigRejectsInvalidPackID(t *testing.T) {
	dir := t.TempDir()

	path := writeConfig(t, dir, `
	[pack]
	id = "Example-Pack"

	[minecraft]
	versions = ">=1.21 <=1.21.6"
	`)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadConfigRequiresMinecraftVersions(t *testing.T) {
	dir := t.TempDir()

	path := writeConfig(t, dir, `
	[pack]
	id = "example"
	`)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindConfigPath(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "src", "nested")

	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}

	expectedPath := writeConfig(t, root, `
	[pack]
	id = "example"

	[minecraft]
	versions = ">=1.21 <=1.21.6"
	`)

	path, err := FindConfigPath(child)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if path != expectedPath {
		t.Fatalf("expected path %q, got %q", expectedPath, path)
	}
}

func TestFindConfigPathNotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := FindConfigPath(dir)
	if !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestLoadNearestConfig(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "src", "nested")

	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}

	writeConfig(t, root, `
	[pack]
	id = "example"

	[minecraft]
	versions = ">=1.21 <=1.21.6"
	`)

	config, projectDir, err := LoadNearestConfig(child)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if projectDir != root {
		t.Fatalf("expected project dir %q, got %q", root, projectDir)
	}

	if config.Pack.ID != "example" {
		t.Fatalf("expected pack id %q, got %q", "example", config.Pack.ID)
	}
}
