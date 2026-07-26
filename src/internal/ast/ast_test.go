package ast

import (
	"reflect"
	"testing"

	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

var _ Assignable = (*VariableExpr)(nil)

func TestOnlyAssignableExpressionsImplementAssignable(t *testing.T) {
	targetField, ok := reflect.TypeOf(AddStmt{}).FieldByName("Target")
	if !ok || targetField.Type != reflect.TypeOf((*Assignable)(nil)).Elem() {
		t.Fatalf("AddStmt.Target must use Assignable, got %v", targetField.Type)
	}

	var expression Expression = &IntLiteral{Value: 10}
	if _, ok := expression.(Assignable); ok {
		t.Fatal("integer literals must not be assignable")
	}

	expression = &VariableExpr{Name: Identifier{Name: "coins"}}
	if _, ok := expression.(Assignable); !ok {
		t.Fatal("variables must be assignable")
	}

	expression = &AccessExpr{Tokens: []token.Token{{Type: token.Ident, Lexeme: "coins"}}}
	if _, ok := expression.(Assignable); !ok {
		t.Fatal("access expressions must be assignable")
	}
}

func TestLeafExpressionsRepresentNilAndPatterns(t *testing.T) {
	var nilExpression Expression = &NilLiteral{}
	if _, ok := nilExpression.(*NilLiteral); !ok {
		t.Fatalf("expected nil literal, got %T", nilExpression)
	}

	pattern := &PatternExpr{Tokens: []token.Token{
		{Type: token.Ident, Lexeme: "coins"},
		{Type: token.Ident, Lexeme: "of"},
		{Type: token.Ident, Lexeme: "player"},
	}}
	var patternExpression Expression = pattern
	got, ok := patternExpression.(*PatternExpr)
	if !ok || len(got.Tokens) != 3 || got.Tokens[0].Lexeme != "coins" || got.Tokens[2].Lexeme != "player" {
		t.Fatalf("unexpected pattern expression: %#v", patternExpression)
	}
}

func TestRequireIsNotATopLevelDeclaration(t *testing.T) {
	var node Node = &RequireDecl{}
	if _, ok := node.(Declaration); ok {
		t.Fatal("require declarations must remain separate from top-level declarations")
	}
}

func TestFileCarriesMetadataRequiresAndTopLevelDeclarations(t *testing.T) {
	path := &StringExpr{Parts: []StringPart{&StringText{Value: "abc/shop"}}}
	alias := &Identifier{Name: "shop"}
	global := &GlobalAssignment{
		Public: true,
		Target: &VariableExpr{Name: Identifier{Name: "tax"}},
		Value:  &FloatLiteral{Value: 0.1},
	}
	function := &FunctionDecl{
		Public: true,
		Name:   Identifier{Name: "price"},
		Parameters: []Parameter{{
			Name: Identifier{Name: "value"},
			Type: &TypeRef{Name: Identifier{Name: "float"}},
		}},
		ReturnType: &TypeRef{Name: Identifier{Name: "float"}},
	}
	event := &EventDecl{Name: []Identifier{{Name: "scoreboard"}, {Name: "update"}}}

	file := File{
		Metadata: []MetadataEntry{
			{Key: "namespace", Value: "example"},
			{Key: "tags", Value: "load, tick"},
		},
		Requirements: []*RequireDecl{{Path: path, Alias: alias}},
		Declarations: []Declaration{global, function, event},
	}

	if len(file.Metadata) != 2 || file.Metadata[0].Key != "namespace" || file.Metadata[0].Value != "example" {
		t.Fatalf("unexpected metadata: %#v", file.Metadata)
	}
	if len(file.Requirements) != 1 || file.Requirements[0].Path != path || file.Requirements[0].Alias.Name != "shop" {
		t.Fatalf("unexpected requirements: %#v", file.Requirements)
	}
	if len(file.Declarations) != 3 {
		t.Fatalf("expected three declarations, got %d", len(file.Declarations))
	}

	gotGlobal, ok := file.Declarations[0].(*GlobalAssignment)
	if !ok || !gotGlobal.Public || gotGlobal.Target.Name.Name != "tax" {
		t.Fatalf("unexpected global declaration: %#v", file.Declarations[0])
	}
	gotFunction, ok := file.Declarations[1].(*FunctionDecl)
	if !ok || !gotFunction.Public || gotFunction.Name.Name != "price" || gotFunction.Parameters[0].Type.Name.Name != "float" {
		t.Fatalf("unexpected function declaration: %#v", file.Declarations[1])
	}
	gotEvent, ok := file.Declarations[2].(*EventDecl)
	if !ok || gotEvent.Name[0].Name != "scoreboard" || gotEvent.Name[1].Name != "update" {
		t.Fatalf("unexpected event declaration: %#v", file.Declarations[2])
	}
}

func TestTypeRefRepresentsNestedGenericTypes(t *testing.T) {
	typeRef := TypeRef{
		Name: Identifier{Name: "map"},
		Arguments: []*TypeRef{
			{Name: Identifier{Name: "string"}},
			{
				Name:      Identifier{Name: "list"},
				Arguments: []*TypeRef{{Name: Identifier{Name: "int"}}},
			},
		},
	}

	if typeRef.Name.Name != "map" || len(typeRef.Arguments) != 2 {
		t.Fatalf("unexpected outer type: %#v", typeRef)
	}
	if typeRef.Arguments[0].Name.Name != "string" {
		t.Fatalf("unexpected key type: %#v", typeRef.Arguments[0])
	}
	listType := typeRef.Arguments[1]
	if listType.Name.Name != "list" || len(listType.Arguments) != 1 || listType.Arguments[0].Name.Name != "int" {
		t.Fatalf("unexpected value type: %#v", listType)
	}
}

func TestStatementsRepresentDocumentedForms(t *testing.T) {
	conditionA := &BoolLiteral{Value: true}
	conditionB := &BoolLiteral{Value: false}
	elseBlock := &Block{}
	ifStatement := &IfStmt{
		Condition: conditionA,
		Then:      Block{Statements: []Statement{&StopStmt{}}},
		ElseIf:    []ElseIfClause{{Condition: conditionB}},
		Else:      elseBlock,
	}

	if ifStatement.Condition != conditionA || len(ifStatement.Then.Statements) != 1 {
		t.Fatalf("unexpected if branch: %#v", ifStatement)
	}
	if len(ifStatement.ElseIf) != 1 || ifStatement.ElseIf[0].Condition != conditionB || ifStatement.Else != elseBlock {
		t.Fatalf("unexpected else branches: %#v", ifStatement)
	}

	value := &IntLiteral{Value: 3}
	target := &VariableExpr{Name: Identifier{Name: "coins"}}
	statements := []Statement{
		&AssignmentStmt{Target: target, Value: value},
		&AddStmt{Value: value, Target: target},
		ifStatement,
		&LoopTimesStmt{Count: value},
		&LoopRangeStmt{Start: &IntLiteral{Value: 1}, End: value},
		&LoopPlayersStmt{},
		&LoopEntitiesStmt{Radius: &IntLiteral{Value: 10}, Around: &CallExpr{Callee: QualifiedName{Parts: []Identifier{{Name: "player"}}}}},
		&ReturnStmt{Value: target},
		&ReturnStmt{},
		&StopStmt{},
		&EffectStmt{Tokens: []token.Token{{Type: token.Ident, Lexeme: "send"}}},
		&ExprStmt{Expression: target},
	}
	wantTypes := []reflect.Type{
		reflect.TypeOf(&AssignmentStmt{}),
		reflect.TypeOf(&AddStmt{}),
		reflect.TypeOf(&IfStmt{}),
		reflect.TypeOf(&LoopTimesStmt{}),
		reflect.TypeOf(&LoopRangeStmt{}),
		reflect.TypeOf(&LoopPlayersStmt{}),
		reflect.TypeOf(&LoopEntitiesStmt{}),
		reflect.TypeOf(&ReturnStmt{}),
		reflect.TypeOf(&ReturnStmt{}),
		reflect.TypeOf(&StopStmt{}),
		reflect.TypeOf(&EffectStmt{}),
		reflect.TypeOf(&ExprStmt{}),
	}

	for index, statement := range statements {
		if reflect.TypeOf(statement) != wantTypes[index] {
			t.Fatalf("statement %d: expected %v, got %T", index, wantTypes[index], statement)
		}
	}
}

func TestExpressionsPreserveSyntaxTreeStructure(t *testing.T) {
	multiply := &BinaryExpr{
		Left:     &IntLiteral{Value: 2},
		Operator: token.Star,
		Right:    &IntLiteral{Value: 3},
	}
	add := &BinaryExpr{
		Left:     &IntLiteral{Value: 1},
		Operator: token.Plus,
		Right:    multiply,
	}
	negated := &UnaryExpr{Operator: token.Not, Operand: &GroupExpr{Expression: add}}
	call := &CallExpr{
		Callee:         QualifiedName{Parts: []Identifier{{Name: "shop"}, {Name: "finalPrice"}}},
		Arguments:      []Expression{negated},
		ExplicitParens: true,
	}

	if add.Operator != token.Plus || add.Right != multiply || multiply.Operator != token.Star {
		t.Fatalf("unexpected binary tree: %#v", add)
	}
	group, ok := negated.Operand.(*GroupExpr)
	if !ok || group.Expression != add {
		t.Fatalf("unexpected unary/group expression: %#v", negated)
	}
	if call.Callee.Parts[0].Name != "shop" || call.Callee.Parts[1].Name != "finalPrice" {
		t.Fatalf("unexpected qualified name: %#v", call.Callee)
	}
	if !call.ExplicitParens || len(call.Arguments) != 1 || call.Arguments[0] != negated {
		t.Fatalf("unexpected call: %#v", call)
	}
}

func TestCollectionAndRangeExpressionsPreserveElements(t *testing.T) {
	one := &IntLiteral{Value: 1}
	two := &IntLiteral{Value: 2}
	list := &ListExpr{Elements: []Expression{one, two}}
	mapExpression := &MapExpr{Entries: []MapEntry{{
		Key:   &StringExpr{Parts: []StringPart{&StringText{Value: "coins"}}},
		Value: two,
	}}}
	rangeExpression := &RangeExpr{Start: one, End: two}

	if len(list.Elements) != 2 || list.Elements[0] != one || list.Elements[1] != two {
		t.Fatalf("unexpected list: %#v", list)
	}
	if len(mapExpression.Entries) != 1 || mapExpression.Entries[0].Value != two {
		t.Fatalf("unexpected map: %#v", mapExpression)
	}
	key, ok := mapExpression.Entries[0].Key.(*StringExpr)
	if !ok || key.Parts[0].(*StringText).Value != "coins" {
		t.Fatalf("unexpected map key: %#v", mapExpression.Entries[0].Key)
	}
	if rangeExpression.Start != one || rangeExpression.End != two {
		t.Fatalf("unexpected range: %#v", rangeExpression)
	}
}

func TestStringsAndVariablesPreserveStructuredParts(t *testing.T) {
	coins := &VariableExpr{
		Qualifier: &Identifier{Name: "shop"},
		Name:      Identifier{Name: "player"},
		Accesses: []VariableAccess{
			&FieldAccess{Field: Identifier{Name: "stats"}},
			&IndexAccess{Index: &VariableExpr{Name: Identifier{Name: "index"}, Local: true}},
			&EmptyIndexAccess{},
		},
	}
	stringExpression := &StringExpr{
		Quote: '"',
		Parts: []StringPart{
			&StringText{Raw: `Coins: {{`, Value: "Coins: {"},
			&StringInterpolation{Expression: coins},
			&StringText{Raw: `}}`, Value: "}"},
		},
	}

	if stringExpression.Quote != '"' || len(stringExpression.Parts) != 3 {
		t.Fatalf("unexpected string: %#v", stringExpression)
	}
	if stringExpression.Parts[0].(*StringText).Value != "Coins: {" || stringExpression.Parts[2].(*StringText).Value != "}" {
		t.Fatalf("unexpected decoded text: %#v", stringExpression.Parts)
	}
	interpolation := stringExpression.Parts[1].(*StringInterpolation)
	if interpolation.Expression != coins {
		t.Fatalf("unexpected interpolation: %#v", interpolation)
	}
	if coins.Qualifier.Name != "shop" || coins.Name.Name != "player" || coins.Local || len(coins.Accesses) != 3 {
		t.Fatalf("unexpected variable: %#v", coins)
	}
	if coins.Accesses[0].(*FieldAccess).Field.Name != "stats" {
		t.Fatalf("unexpected field access: %#v", coins.Accesses[0])
	}
	if !coins.Accesses[1].(*IndexAccess).Index.(*VariableExpr).Local {
		t.Fatalf("expected local index variable: %#v", coins.Accesses[1])
	}
}

func TestSpanHelpersPreserveSourceCoordinates(t *testing.T) {
	first := diagnostic.Span{
		StartLine: 2, StartColumn: 3, EndLine: 2, EndColumn: 4,
		StartOffset: 5, EndOffset: 6,
	}
	last := diagnostic.Span{
		StartLine: 4, StartColumn: 1, EndLine: 4, EndColumn: 8,
		StartOffset: 20, EndOffset: 27,
	}
	want := diagnostic.Span{
		StartLine: 2, StartColumn: 3, EndLine: 4, EndColumn: 8,
		StartOffset: 5, EndOffset: 27,
	}

	if got := JoinSpans(first, last); got != want {
		t.Fatalf("expected joined span %#v, got %#v", want, got)
	}

	left := &IntLiteral{NodeBase: NodeBase{SourceSpan: first}, Value: 1}
	right := &IntLiteral{NodeBase: NodeBase{SourceSpan: last}, Value: 2}
	if got := SpanBetween(left, right); got != want {
		t.Fatalf("expected node span %#v, got %#v", want, got)
	}
	if left.Span() != first || right.Span() != last {
		t.Fatalf("span propagation changed child spans: left=%#v right=%#v", left.Span(), right.Span())
	}
}
