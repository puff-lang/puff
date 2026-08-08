package lower

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/ir"
	"github.com/puff-lang/puff/internal/lexer"
	"github.com/puff-lang/puff/internal/parser"
	"github.com/puff-lang/puff/internal/sema"
	"github.com/puff-lang/puff/internal/source"
)

func readTestdata(t *testing.T, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return strings.ReplaceAll(string(content), "\r\n", "\n")
}

func checkedProject(t *testing.T, sources map[string]string, order ...string) *sema.Project {
	t.Helper()

	if len(order) == 0 {
		for relPath := range sources {
			order = append(order, relPath)
		}
		sort.Strings(order)
	}

	project := &sema.Project{Root: "testdata", Modules: make([]*sema.Module, 0, len(order))}
	for _, relPath := range order {
		text, ok := sources[relPath]
		if !ok {
			t.Fatalf("missing source for %q", relPath)
		}
		file := source.NewFile(relPath, relPath, text)
		parsed := parser.Parse(file, lexer.Lex(file))
		if len(parsed.Diagnostics) != 0 {
			t.Fatalf("parse %s: %#v", relPath, parsed.Diagnostics)
		}
		project.Modules = append(project.Modules, &sema.Module{
			Source:  file,
			Syntax:  parsed.File,
			Imports: make(map[string]*sema.Import),
		})
	}

	checked := sema.Check(project)
	if len(checked.Diagnostics) != 0 {
		t.Fatalf("check project: %#v", checked.Diagnostics)
	}
	return checked.Project
}

func parseAST(t *testing.T, relPath string, text string) *ast.File {
	t.Helper()

	file := source.NewFile(relPath, relPath, text)
	parsed := parser.Parse(file, lexer.Lex(file))
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("parse %s: %#v", relPath, parsed.Diagnostics)
	}
	return parsed.File
}

func renderAST(relPath string, file *ast.File) string {
	var out strings.Builder
	fmt.Fprintf(&out, "file %s\n", relPath)
	for _, entry := range file.Metadata {
		fmt.Fprintf(&out, "metadata %s=%s\n", entry.Key, entry.Value)
	}
	for _, declaration := range file.Declarations {
		switch node := declaration.(type) {
		case *ast.GlobalAssignment:
			fmt.Fprintf(&out, "global %s = %s\n", renderASTVariable(node.Target), renderASTExpression(node.Value))
		case *ast.FunctionDecl:
			fmt.Fprintf(&out, "function %s() -> %s\n", node.Name.Name, renderASTType(node.ReturnType))
			renderASTBlock(&out, node.Body)
		case *ast.EventDecl:
			names := make([]string, len(node.Name))
			for index, name := range node.Name {
				names[index] = name.Name
			}
			fmt.Fprintf(&out, "event %s\n", strings.Join(names, " "))
			renderASTBlock(&out, node.Body)
		default:
			fmt.Fprintf(&out, "declaration %T\n", declaration)
		}
	}
	return out.String()
}

func renderASTBlock(out *strings.Builder, block ast.Block) {
	for _, statement := range block.Statements {
		switch node := statement.(type) {
		case *ast.ReturnStmt:
			fmt.Fprintf(out, "  return %s\n", renderASTExpression(node.Value))
		case *ast.EffectStmt:
			out.WriteString("  effect")
			for _, current := range node.Tokens {
				fmt.Fprintf(out, " %s(%s)", current.Type, strconv.Quote(current.Lexeme))
			}
			out.WriteByte('\n')
		default:
			fmt.Fprintf(out, "  statement %T\n", statement)
		}
	}
}

func renderASTExpression(expression ast.Expression) string {
	switch node := expression.(type) {
	case *ast.IntLiteral:
		return fmt.Sprintf("int(%d)", node.Value)
	case *ast.StringExpr:
		parts := make([]string, 0, len(node.Parts))
		for _, part := range node.Parts {
			switch current := part.(type) {
			case *ast.StringText:
				parts = append(parts, "literal("+strconv.Quote(current.Value)+")")
			case *ast.StringInterpolation:
				parts = append(parts, "interpolation("+renderASTExpression(current.Expression)+")")
			}
		}
		return "text[" + strings.Join(parts, ", ") + "]"
	case *ast.CallExpr:
		parts := make([]string, len(node.Callee.Parts))
		for index, part := range node.Callee.Parts {
			parts[index] = part.Name
		}
		return "call(" + strings.Join(parts, ".") + ")"
	default:
		return fmt.Sprintf("%T", expression)
	}
}

func renderASTVariable(variable *ast.VariableExpr) string {
	if variable == nil {
		return "<nil>"
	}
	return "$" + variable.Name.Name
}

func renderASTType(ref *ast.TypeRef) string {
	if ref == nil {
		return "nil"
	}
	return ref.Name.Name
}

func renderIR(project *ir.Project) string {
	if project == nil {
		return "<nil project>\n"
	}

	var out strings.Builder
	out.WriteString("project\n")
	fmt.Fprintf(&out, "modules %d\n", len(project.Modules))
	for _, module := range project.Modules {
		fmt.Fprintf(&out, "  module %s namespace=%s source=%s\n", module.Path, module.Namespace, module.Source.File)
	}
	fmt.Fprintf(&out, "globals %d\n", len(project.Globals))
	for _, global := range project.Globals {
		fmt.Fprintf(&out, "  global %s public=%t type=%s = %s\n",
			renderSymbol(global.ID), global.Public, renderIRType(global.Type), renderIRValue(global.Initializer))
	}
	fmt.Fprintf(&out, "functions %d\n", len(project.Functions))
	for _, function := range project.Functions {
		parameters := make([]string, len(function.Parameters))
		for index, parameter := range function.Parameters {
			parameters[index] = parameter.Name + ":" + renderIRType(parameter.Type)
		}
		fmt.Fprintf(&out, "  function %s %s public=%t (%s) -> %s\n",
			function.Kind, renderSymbol(function.ID), function.Public, strings.Join(parameters, ", "), renderIRType(function.Result))
		for _, command := range function.Commands {
			renderIRCommand(&out, command)
		}
	}
	fmt.Fprintf(&out, "tags %d\n", len(project.Tags))
	for _, tag := range project.Tags {
		entries := make([]string, len(tag.Functions))
		for index, function := range tag.Functions {
			entries[index] = renderSymbol(function)
		}
		fmt.Fprintf(&out, "  tag %s -> %s\n", tag.Name, strings.Join(entries, ", "))
	}
	return out.String()
}

func renderIRCommand(out *strings.Builder, command ir.Command) {
	switch node := command.(type) {
	case *ir.Return:
		fmt.Fprintf(out, "    return %s\n", renderIRValue(node.Value))
	case *ir.Effect:
		fmt.Fprintf(out, "    effect %s\n", node.PatternID)
		for _, argument := range node.Arguments {
			fmt.Fprintf(out, "      %s = %s\n", argument.Name, renderIRValue(argument.Value))
		}
	default:
		fmt.Fprintf(out, "    command %T\n", command)
	}
}

func renderIRValue(value ir.Value) string {
	switch node := value.(type) {
	case *ir.Nil:
		return "nil"
	case *ir.Bool:
		return fmt.Sprintf("bool(%t)", node.Value)
	case *ir.Int:
		return fmt.Sprintf("int(%d)", node.Value)
	case *ir.Float:
		return fmt.Sprintf("float(%s)", strconv.FormatFloat(node.Value, 'g', -1, 64))
	case *ir.Text:
		parts := make([]string, len(node.Parts))
		for index, part := range node.Parts {
			switch current := part.(type) {
			case *ir.TextLiteral:
				parts[index] = "literal(" + strconv.Quote(current.Value) + ")"
			case *ir.TextInterpolation:
				parts[index] = "interpolation(" + renderIRValue(current.Value) + ")"
			default:
				parts[index] = fmt.Sprintf("%T", part)
			}
		}
		return "text[" + strings.Join(parts, ", ") + "]"
	case *ir.Call:
		arguments := make([]string, len(node.Arguments))
		for index, argument := range node.Arguments {
			arguments[index] = renderIRValue(argument)
		}
		if len(arguments) == 0 {
			return "call(" + renderSymbol(node.Function) + ")"
		}
		return "call(" + renderSymbol(node.Function) + ", " + strings.Join(arguments, ", ") + ")"
	case *ir.Reference:
		return "reference(" + node.Name + ":" + renderIRType(node.Type) + ")"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func renderSymbol(id ir.SymbolID) string {
	if id.Module == "" {
		return id.Name
	}
	return id.Module + "::" + id.Name
}

func renderIRType(typ ir.Type) string {
	if typ.Kind == ir.TypeNamed {
		return "named(" + typ.Name + ")"
	}
	if len(typ.Arguments) == 0 {
		return string(typ.Kind)
	}
	arguments := make([]string, len(typ.Arguments))
	for index, argument := range typ.Arguments {
		arguments[index] = renderIRType(argument)
	}
	return string(typ.Kind) + "<" + strings.Join(arguments, ", ") + ">"
}

func findEventFunction(t *testing.T, project *ir.Project, event string) *ir.Function {
	t.Helper()

	for index := range project.Functions {
		function := &project.Functions[index]
		if function.Kind == ir.FunctionEvent && strings.Contains(function.ID.Name, event) {
			return function
		}
	}
	t.Fatalf("missing %s event function in %#v", event, project.Functions)
	return nil
}

func effectArgument(t *testing.T, effect *ir.Effect, name string) ir.Value {
	t.Helper()

	for _, argument := range effect.Arguments {
		if argument.Name == name {
			return argument.Value
		}
	}
	t.Fatalf("missing %q argument in %#v", name, effect.Arguments)
	return nil
}
