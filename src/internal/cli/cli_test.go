package cli_test

import (
	"bytes"
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
	output, err := executeCommand("bundle", "--target", "1.21.6", "--output", "dist")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "bundle --target 1.21.6 --output dist\n"
	if output != expected {
		t.Fatalf("expected %q, got %q", expected, output)
	}
}
