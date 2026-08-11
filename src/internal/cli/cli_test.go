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
	root := writeCheckProject(t, "on load\nend\n")
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}
	t.Chdir(nested)

	output, err := executeCommand("check")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output != "" {
		t.Fatalf("expected no output, got %q", output)
	}
	if _, err := os.Stat(filepath.Join(root, "dist")); !os.IsNotExist(err) {
		t.Fatalf("check created dist: %v", err)
	}
}

func TestBundleCommand(t *testing.T) {
	output, err := executeCommand("bundle", "--target", "1.21.6", "--output", "dist")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "bundle --target 1.21.6 --output dist\n"
	if output != expected {
		t.Fatalf("expected %q, got %q", expected, output)
	}
}

func writeCheckProject(t *testing.T, source string) string {
	t.Helper()

	root := t.TempDir()
	writeCheckFile(t, filepath.Join(root, "puff.toml"), `[pack]
id = "cli-check"
name = "CLI Check"

[minecraft]
versions = ">=1.21 <=1.21.6"
target = "1.21.6"
`)
	writeCheckFile(t, filepath.Join(root, "src", "main.puff"), source)
	return root
}

func writeCheckFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
