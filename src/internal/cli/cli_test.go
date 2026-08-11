package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/puff-lang/puff/internal/cli"
)

func executeCommand(args ...string) (string, string, error) {
	cmd := cli.NewRootCommand()

	var out bytes.Buffer
	var errOut bytes.Buffer

	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestVersionCommand(t *testing.T) {
	output, errOutput, err := executeCommand("version")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if errOutput != "" {
		t.Fatalf("expected no error output, got %q", errOutput)
	}

	pattern := regexp.MustCompile(`(?m)^puff\s+\S+\ncommit:\s+\S+\ndate:\s+\S+\n$`)

	if !pattern.MatchString(output) {
		t.Fatalf("expected version output, got %q", output)
	}
}

func TestInitCommand(t *testing.T) {
	output, errOutput, err := executeCommand("init", "example")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if errOutput != "" {
		t.Fatalf("expected no error output, got %q", errOutput)
	}

	expected := "init example\n"
	if output != expected {
		t.Fatalf("expected %q, got %q", expected, output)
	}
}

func TestCheckCommand(t *testing.T) {
	output, errOutput, err := executeCommand("check")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if errOutput != "" {
		t.Fatalf("expected no error output, got %q", errOutput)
	}

	expected := "check\n"
	if output != expected {
		t.Fatalf("expected %q, got %q", expected, output)
	}
}

func TestBundleCommand(t *testing.T) {
	root := writeBundleProject(t, bundleValidSource)
	t.Chdir(root)

	output, errOutput, err := executeCommand("bundle")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output != "" {
		t.Fatalf("expected no output, got %q", output)
	}
	if errOutput != "" {
		t.Fatalf("expected no error output, got %q", errOutput)
	}
	assertBundleMetadata(t, filepath.Join(root, "configured-dist"), 80)
	if _, err := os.Stat(filepath.Join(root, "dist")); !os.IsNotExist(err) {
		t.Fatalf("bundle ignored configured output: %v", err)
	}
}

func TestBundleCommandWritesDefaultOutput(t *testing.T) {
	root := writeDefaultBundleProject(t, bundleValidSource)
	t.Chdir(root)

	output, errOutput, err := executeCommand("bundle")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output != "" || errOutput != "" {
		t.Fatalf("expected no output, got stdout %q and stderr %q", output, errOutput)
	}
	assertBundleMetadata(t, filepath.Join(root, "dist"), 80)
}

func TestBundleCommandOverridesOutput(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{name: "long flag", flag: "--output"},
		{name: "short flag", flag: "-o"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := writeBundleProject(t, bundleValidSource)
			t.Chdir(root)

			customOutput := filepath.Join("custom", "datapack")
			output, errOutput, err := executeCommand("bundle", test.flag, customOutput)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if output != "" || errOutput != "" {
				t.Fatalf("expected no output, got stdout %q and stderr %q", output, errOutput)
			}
			assertBundleMetadata(t, filepath.Join(root, customOutput), 80)
			if _, err := os.Stat(filepath.Join(root, "configured-dist")); !os.IsNotExist(err) {
				t.Fatalf("bundle wrote configured output despite override: %v", err)
			}
		})
	}
}

func TestBundleCommandPassesTarget(t *testing.T) {
	root := writeBundleProject(t, bundleValidSource)
	t.Chdir(root)

	output, errOutput, err := executeCommand("bundle", "--target", "1.21.5")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output != "" || errOutput != "" {
		t.Fatalf("expected no output, got stdout %q and stderr %q", output, errOutput)
	}
	assertBundleMetadata(t, filepath.Join(root, "configured-dist"), 71)
}

func TestBundleCommandPreservesOutputOnDiagnostics(t *testing.T) {
	root := writeBundleProject(t, "# tags: load\n\non tick\nend\n")
	outputDir := filepath.Join(root, "configured-dist")
	sentinel := filepath.Join(outputDir, "sentinel.txt")
	writeBundleFile(t, sentinel, "keep me\n")
	t.Chdir(root)

	output, errOutput, err := executeCommand("bundle")
	if err == nil {
		t.Fatal("expected bundle to fail")
	}
	if output != "" {
		t.Fatalf("expected no standard output, got %q", output)
	}
	expected := "error[MISSING_LOAD_EVENT]: Missing required event: on load\n" +
		"  --> main.puff:1:1\n" +
		"   |\n" +
		"   = hint: Add an on load block or remove the load tag.\n"
	if errOutput != expected {
		t.Fatalf("expected %q, got %q", expected, errOutput)
	}
	data, readErr := os.ReadFile(sentinel)
	if readErr != nil {
		t.Fatalf("read preserved output: %v", readErr)
	}
	if string(data) != "keep me\n" {
		t.Fatalf("preserved output = %q, want %q", data, "keep me\n")
	}
	entries, readDirErr := os.ReadDir(outputDir)
	if readDirErr != nil {
		t.Fatalf("read preserved output directory: %v", readDirErr)
	}
	if len(entries) != 1 || entries[0].Name() != "sentinel.txt" || entries[0].IsDir() {
		t.Fatalf("output entries = %#v, want only sentinel.txt", entries)
	}
}

func TestBundleCommandReportsMissingPuffTOML(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	output, errOutput, err := executeCommand("bundle")
	if err == nil {
		t.Fatal("expected bundle to fail")
	}
	if output != "" {
		t.Fatalf("expected no standard output, got %q", output)
	}
	expected := "error[MISSING_PUFF_TOML]: Missing puff.toml.\n" +
		"  --> <unknown>:1:1\n" +
		"   |\n" +
		"   = hint: Run puff init to create a new project.\n"
	if errOutput != expected {
		t.Fatalf("expected %q, got %q", expected, errOutput)
	}
	if _, statErr := os.Stat(filepath.Join(root, "dist")); !os.IsNotExist(statErr) {
		t.Fatalf("failed bundle created dist: %v", statErr)
	}
}

func TestBundleCommandRejectsArguments(t *testing.T) {
	root := writeBundleProject(t, bundleValidSource)
	t.Chdir(root)

	output, errOutput, err := executeCommand("bundle", "unexpected")
	if err == nil {
		t.Fatal("expected bundle to reject arguments")
	}
	if output != "" || errOutput != "" {
		t.Fatalf("expected no output, got stdout %q and stderr %q", output, errOutput)
	}
	if _, statErr := os.Stat(filepath.Join(root, "configured-dist")); !os.IsNotExist(statErr) {
		t.Fatalf("bundle created output despite invalid arguments: %v", statErr)
	}
}

func TestBundleCommandRejectsRemovedAllTargetsFlag(t *testing.T) {
	root := writeBundleProject(t, bundleValidSource)
	t.Chdir(root)

	output, errOutput, err := executeCommand("bundle", "--all-targets")
	if err == nil {
		t.Fatal("expected bundle to reject --all-targets")
	}
	if output != "" || errOutput != "" {
		t.Fatalf("expected no output, got stdout %q and stderr %q", output, errOutput)
	}
	if _, statErr := os.Stat(filepath.Join(root, "configured-dist")); !os.IsNotExist(statErr) {
		t.Fatalf("bundle created output despite unsupported flag: %v", statErr)
	}
}

const bundleValidSource = "# tags: load\n\non load\nend\n"

const bundleConfig = `[pack]
id = "cli_bundle"
name = "CLI Bundle"

[minecraft]
versions = ">=1.21 <=1.21.6"
target = "1.21.6"

[build]
source = "src"
output = "configured-dist"
`

const bundleDefaultConfig = `[pack]
id = "cli_bundle"
name = "CLI Bundle"

[minecraft]
versions = ">=1.21 <=1.21.6"
target = "1.21.6"
`

func writeBundleProject(t *testing.T, source string) string {
	t.Helper()
	return writeBundleProjectWithConfig(t, bundleConfig, source)
}

func writeDefaultBundleProject(t *testing.T, source string) string {
	t.Helper()
	return writeBundleProjectWithConfig(t, bundleDefaultConfig, source)
}

func writeBundleProjectWithConfig(t *testing.T, config string, source string) string {
	t.Helper()

	root := t.TempDir()
	writeBundleFile(t, filepath.Join(root, "puff.toml"), config)
	writeBundleFile(t, filepath.Join(root, "src", "main.puff"), source)
	return root
}

func assertBundleMetadata(t *testing.T, output string, packFormat int) {
	t.Helper()

	metadata, err := os.ReadFile(filepath.Join(output, "pack.mcmeta"))
	if err != nil {
		t.Fatalf("read generated pack.mcmeta: %v", err)
	}
	expected := "{\"pack\":{\"pack_format\":" + strconv.Itoa(packFormat) + ",\"description\":\"CLI Bundle\"}}\n"
	if string(metadata) != expected {
		t.Fatalf("expected %q, got %q", expected, metadata)
	}
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
