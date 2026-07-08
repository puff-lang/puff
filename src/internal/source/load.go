package source

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/puff-lang/puff/internal/project"
)

var ErrSourceDirNotFound = errors.New("source directory not found")

type Project struct {
	Root  string
	Files []File
}

func LoadFile(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read source file: %w", err)
	}

	return NewFile(path, filepath.Base(path), string(data)), nil
}

func LoadProject(root string, config project.Config) (Project, error) {
	sourceDir := filepath.Join(root, sourceDirName(config))

	info, err := os.Stat(sourceDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Project{}, fmt.Errorf("%w: %s", ErrSourceDirNotFound, sourceDir)
		}

		return Project{}, fmt.Errorf("check source directory: %w", err)
	}
	if !info.IsDir() {
		return Project{}, fmt.Errorf("source path is not a directory: %s", sourceDir)
	}

	var files []File

	err = filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk source directory: %w", walkErr)
		}
		if entry.IsDir() || filepath.Ext(path) != ".puff" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read source file: %w", err)
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("resolve source relative path: %w", err)
		}

		files = append(files, NewFile(path, filepath.ToSlash(relPath), string(data)))

		return nil
	})
	if err != nil {
		return Project{}, err
	}

	sort.Slice(files, func(i int, j int) bool {
		return files[i].RelPath < files[j].RelPath
	})

	return Project{
		Root:  root,
		Files: files,
	}, nil
}

func sourceDirName(config project.Config) string {
	if config.Build.Source == "" {
		return project.DefaultSourceDir
	}

	return config.Build.Source
}
