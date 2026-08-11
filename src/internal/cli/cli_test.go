package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/puff-lang/puff/internal/cli"
)

func executeCommand(args ...string) (string, error) {
	cmd := cli.NewRootCommand()

	var out bytes.Buffer
	var errOut bytes.Buffer

	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err := cmd.Execute()

	if errOut.Len() > 0 {
		return out.String() + errOut.String(), err
	}

	return out.String(), err
}

func TestVersionCommand(t *testing.T) {
	output, err := executeCommand("version")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	pattern := regexp.MustCompile(`(?m)^puff\s+\S+\ncommit:\s+\S+\ndate:\s+\S+\n$`)

	if !pattern.MatchString(output) {
		t.Fatalf("expected version output, got %q", output)
	}
}

func TestInitCommand(t *testing.T) {
	output, err := executeCommand("init", "example")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "init example\n"
	if output != expected {
		t.Fatalf("expected %q, got %q", expected, output)
	}
}

func TestCheckCommand(t *testing.T) {
	output, err := executeCommand("check")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "check\n"
	if output != expected {
		t.Fatalf("expected %q, got %q", expected, output)
	}
}

func TestBundleCommand(t *testing.T) {
	root := writeBundleProject(t, bundleValidSource)
	t.Chdir(root)

	output, err := executeCommand("bundle")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output != "" {
		t.Fatalf("expected no output, got %q", output)
	}

	metadata, err := os.ReadFile(filepath.Join(root, "configured-dist", "pack.mcmeta"))
	if err != nil {
		t.Fatalf("read generated pack.mcmeta: %v", err)
	}
	expected := "{\"pack\":{\"pack_format\":80,\"description\":\"CLI Bundle\"}}\n"
	if string(metadata) != expected {
		t.Fatalf("expected %q, got %q", expected, metadata)
	}
	if _, err := os.Stat(filepath.Join(root, "dist")); !os.IsNotExist(err) {
		t.Fatalf("bundle ignored configured output: %v", err)
	}
}

const bundleValidSource = "# tags: load\n\non load\nend\n"

func writeBundleProject(t *testing.T, source string) string {
	t.Helper()

	root := t.TempDir()
	writeBundleFile(t, filepath.Join(root, "puff.toml"), `[pack]
id = "cli_bundle"
name = "CLI Bundle"

[minecraft]
versions = ">=1.21 <=1.21.6"
target = "1.21.6"

[build]
source = "src"
output = "configured-dist"
`)
	writeBundleFile(t, filepath.Join(root, "src", "main.puff"), source)
	return root
}

func writeBundleFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
