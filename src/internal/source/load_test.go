package source

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/puff-lang/puff/internal/project"
)

func TestLoadFileReadsPuffSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.puff")

	if err := os.WriteFile(path, []byte("on load\nend\n"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	file, err := LoadFile(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if file.Path != path {
		t.Fatalf("expected path %q, got %q", path, file.Path)
	}
	if file.RelPath != "main.puff" {
		t.Fatalf("expected rel path %q, got %q", "main.puff", file.RelPath)
	}
	if file.Text != "on load\nend\n" {
		t.Fatalf("expected file text %q, got %q", "on load\nend\n", file.Text)
	}
}

func TestLoadProjectReadsPuffFilesRecursively(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "src")
	nestedDir := filepath.Join(sourceDir, "nested")

	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "main.puff"), []byte("on load\nend\n"), 0644); err != nil {
		t.Fatalf("failed to write main source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "shop.puff"), []byte("fun price\nend\n"), 0644); err != nil {
		t.Fatalf("failed to write nested source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "notes.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatalf("failed to write ignored file: %v", err)
	}

	loaded, err := LoadProject(root, project.Config{
		Build: project.BuildConfig{
			Source: "src",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if loaded.Root != root {
		t.Fatalf("expected root %q, got %q", root, loaded.Root)
	}
	if len(loaded.Files) != 2 {
		t.Fatalf("expected 2 source files, got %d", len(loaded.Files))
	}
	if loaded.Files[0].RelPath != "main.puff" {
		t.Fatalf("expected first rel path %q, got %q", "main.puff", loaded.Files[0].RelPath)
	}
	if loaded.Files[1].RelPath != "nested/shop.puff" {
		t.Fatalf("expected second rel path %q, got %q", "nested/shop.puff", loaded.Files[1].RelPath)
	}
	if loaded.Files[1].Text != "fun price\nend\n" {
		t.Fatalf("expected nested text %q, got %q", "fun price\nend\n", loaded.Files[1].Text)
	}
}

func TestLoadProjectUsesDefaultSourceDir(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, project.DefaultSourceDir)

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "main.puff"), []byte("on load\nend\n"), 0644); err != nil {
		t.Fatalf("failed to write main source: %v", err)
	}

	loaded, err := LoadProject(root, project.Config{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(loaded.Files) != 1 {
		t.Fatalf("expected 1 source file, got %d", len(loaded.Files))
	}
	if loaded.Files[0].RelPath != "main.puff" {
		t.Fatalf("expected rel path %q, got %q", "main.puff", loaded.Files[0].RelPath)
	}
}

func TestLoadProjectReturnsTypedErrorForMissingSourceDir(t *testing.T) {
	root := t.TempDir()

	_, err := LoadProject(root, project.Config{
		Build: project.BuildConfig{
			Source: "src",
		},
	})
	if !errors.Is(err, ErrSourceDirNotFound) {
		t.Fatalf("expected ErrSourceDirNotFound, got %v", err)
	}
}

func TestLoadProjectRejectsSourcePathFile(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "src")

	if err := os.WriteFile(sourcePath, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("failed to write source path file: %v", err)
	}

	_, err := LoadProject(root, project.Config{
		Build: project.BuildConfig{
			Source: "src",
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
