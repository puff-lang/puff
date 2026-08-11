package minecraft

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Write stages a complete datapack and replaces the previous output tree.
func Write(output Output, outputDir string) error {
	return WriteContext(context.Background(), output, outputDir)
}

// WriteContext stages a complete datapack and stops before publication when canceled.
func WriteContext(ctx context.Context, output Output, outputDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if outputDir == "" {
		return fmt.Errorf("output directory is empty")
	}
	root, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	parent := filepath.Dir(root)
	if parent == root {
		return fmt.Errorf("output directory cannot be a filesystem root")
	}

	files := append([]File(nil), output.Files...)
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		key, err := portablePathKey(file.Path)
		if err != nil {
			return err
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate output path %q", file.Path)
		}
		seen[key] = struct{}{}
	}
	for key := range seen {
		for parentKey := key; ; {
			separator := strings.LastIndexByte(parentKey, '/')
			if separator < 0 {
				break
			}
			parentKey = parentKey[:separator]
			if _, exists := seen[parentKey]; exists {
				return fmt.Errorf("output path %q conflicts with a file", key)
			}
		}
	}
	sort.Slice(files, func(left, right int) bool { return files[left].Path < files[right].Path })

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create output parent %q: %w", parent, err)
	}
	parent, err = filepath.EvalSymlinks(parent)
	if err != nil {
		return fmt.Errorf("resolve output parent %q: %w", parent, err)
	}
	root = filepath.Join(parent, filepath.Base(root))
	if info, err := os.Lstat(root); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("output directory %q is not a directory", root)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect output directory %q: %w", root, err)
	}

	stage, err := os.MkdirTemp(parent, "."+filepath.Base(root)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create staged output: %w", err)
	}
	defer func() {
		if stage != "" {
			_ = os.RemoveAll(stage)
		}
	}()

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		destination := filepath.Join(stage, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return fmt.Errorf("create staged directory for %q: %w", file.Path, err)
		}
		if err := os.WriteFile(destination, file.Data, 0o644); err != nil {
			return fmt.Errorf("write staged output %q: %w", file.Path, err)
		}
	}

	if _, err := os.Lstat(root); os.IsNotExist(err) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := os.Rename(stage, root); err != nil {
			return fmt.Errorf("publish output: %w", err)
		}
		stage = ""
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect output directory %q: %w", root, err)
	}

	backup, err := reserveSiblingPath(parent, "."+filepath.Base(root)+".old-*")
	if err != nil {
		return fmt.Errorf("reserve previous output: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(root, backup); err != nil {
		return fmt.Errorf("move previous output: %w", err)
	}
	if err := ctx.Err(); err != nil {
		if restoreErr := os.Rename(backup, root); restoreErr != nil {
			return fmt.Errorf("cancel output publication: %v; restore previous output from %q: %w", err, backup, restoreErr)
		}
		return err
	}
	if err := os.Rename(stage, root); err != nil {
		if restoreErr := os.Rename(backup, root); restoreErr != nil {
			return fmt.Errorf("publish output: %v; restore previous output from %q: %w", err, backup, restoreErr)
		}
		return fmt.Errorf("publish output: %w", err)
	}
	stage = ""
	_ = os.RemoveAll(backup)
	return nil
}

func portablePathKey(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, `\:`) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("invalid output path %q", name)
	}
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") || strings.ContainsAny(part, `<>"|?*`) || windowsReservedName(part) {
			return "", fmt.Errorf("invalid output path %q", name)
		}
	}
	return strings.ToLower(name), nil
}

func windowsReservedName(name string) bool {
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}

func reserveSiblingPath(parent, pattern string) (string, error) {
	reserved, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return "", err
	}
	if err := os.Remove(reserved); err != nil {
		return "", err
	}
	return reserved, nil
}
