package ir_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/puff-lang/puff/internal/ir"
)

func TestExportedIRGraphDoesNotReferenceParserOrSemanticModels(t *testing.T) {
	forbidden := map[string]bool{
		"github.com/puff-lang/puff/internal/ast":      true,
		"github.com/puff-lang/puff/internal/token":    true,
		"github.com/puff-lang/puff/internal/patterns": true,
		"github.com/puff-lang/puff/internal/sema":     true,
	}
	roots := []reflect.Type{
		reflect.TypeOf(ir.Project{}),
		reflect.TypeOf(ir.Return{}),
		reflect.TypeOf(ir.Effect{}),
		reflect.TypeOf(ir.Nil{}),
		reflect.TypeOf(ir.Bool{}),
		reflect.TypeOf(ir.Int{}),
		reflect.TypeOf(ir.Float{}),
		reflect.TypeOf(ir.Text{}),
		reflect.TypeOf(ir.TextLiteral{}),
		reflect.TypeOf(ir.TextInterpolation{}),
		reflect.TypeOf(ir.Call{}),
		reflect.TypeOf(ir.Reference{}),
	}

	seen := make(map[reflect.Type]bool)
	for _, root := range roots {
		walkTypeGraph(t, root, root.String(), forbidden, seen)
	}
}

func TestIRGoldenContainsNoMinecraftOutput(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "lower", "testdata", "target.ir.golden"))
	if err != nil {
		t.Fatalf("read IR golden: %v", err)
	}
	lowered := strings.ToLower(string(content))
	for _, forbidden := range []string{"tellraw", "@a", ".mcfunction", "data/"} {
		if strings.Contains(lowered, forbidden) {
			t.Errorf("IR golden contains Minecraft output %q", forbidden)
		}
	}
}

func walkTypeGraph(
	t *testing.T,
	current reflect.Type,
	path string,
	forbidden map[string]bool,
	seen map[reflect.Type]bool,
) {
	t.Helper()

	for current.Kind() == reflect.Pointer || current.Kind() == reflect.Slice || current.Kind() == reflect.Array {
		current = current.Elem()
	}
	if forbidden[current.PkgPath()] {
		t.Errorf("IR type graph path %s references forbidden package %s", path, current.PkgPath())
		return
	}
	if seen[current] {
		return
	}
	seen[current] = true

	switch current.Kind() {
	case reflect.Struct:
		for index := 0; index < current.NumField(); index++ {
			field := current.Field(index)
			if field.IsExported() {
				walkTypeGraph(t, field.Type, path+"."+field.Name, forbidden, seen)
			}
		}
	case reflect.Map:
		walkTypeGraph(t, current.Key(), path+"[key]", forbidden, seen)
		walkTypeGraph(t, current.Elem(), path+"[value]", forbidden, seen)
	}
}
