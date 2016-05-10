package parse

import (
	"ast"
	"fmt"
	"runtime"
	"strconv"
	"token"
)

// Tree is the representation of a single parsed template.
type Tree struct {
	Name      string        // name of the template represented by the tree.
	ParseName string        // name of the top-level template during parsing, for error messages.
	Root      *ast.ListNode // top-level root of the tree.
	text      string        // text parsed to create the template (or its parent)
	// Parsing only; cleared after parse.
	funcs     []map[string]interface{}
	lex       *lexer
	token     [3]token.Token // three-token lookahead for parser.
	peekCount int
	vars      []string // variables defined at the moment.
	treeSet   map[string]*Tree

	topScope *ast.Scope // Scope for Global Variables inside the file
}

// Copy returns a copy of the Tree. Any parsing state is discarded.
func (t *Tree) Copy() *Tree {
	if t == nil {
		return nil
	}
	return &Tree{
		Name:      t.Name,
		ParseName: t.ParseName,
		Root:      t.Root.CopyList(),
		text:      t.text,
	}
}

// Parse returns a map from template name to parse.Tree, created by parsing the
// templates described in the argument string. The top-level template will be
// given the specified name. If an error is encountered, parsing stops and an
// empty map is returned with the error.
func Parse(name, text, leftDelim, rightDelim string, funcs ...map[string]interface{}) (treeSet map[string]*Tree, err error) {
	treeSet = make(map[string]*Tree)
	t := New(name)
	t.text = text
	_, err = t.Parse(text, leftDelim, rightDelim, treeSet, funcs...)
	return
}

// next returns the next token.
func (t *Tree) next() token.Token {
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		t.token[0] = t.lex.nextToken()
	}
	return t.token[t.peekCount]
}

// backup backs the input stream up one token.
func (t *Tree) backup() {
	t.peekCount++
}

// backup2 backs the input stream up two tokens.
// The zeroth token is already there.
func (t *Tree) backup2(t1 token.Token) {
	t.token[1] = t1
	t.peekCount = 2
}

// backup3 backs the input stream up three tokens
// The zeroth token is already there.
func (t *Tree) backup3(t2, t1 token.Token) { // Reverse order: we're pushing back.
	t.token[1] = t1
	t.token[2] = t2
	t.peekCount = 3
}

// peek returns but does not consume the next token.
func (t *Tree) peek() token.Token {
	if t.peekCount > 0 {
		return t.token[t.peekCount-1]
	}
	t.peekCount += 1
	t.token[0] = t.lex.nextToken()
	return t.token[0]
}

// nextNonSpace returns the next non-space token.
func (t *Tree) nextNonSpace() (tok token.Token) {
	for {
		tok = t.next()
		if tok.Type() != token.WHITESPACE {
			break
		}
	}
	return tok
}

// peekNonSpace returns but does not consume the next non-space token.
func (t *Tree) peekNonSpace() (tok token.Token) {
	for {
		tok = t.next()
		if tok.Type() != token.WHITESPACE {
			break
		}
	}
	t.backup()
	return tok
}

// Parsing.

// New allocates a new parse tree with the given name.
func New(name string, funcs ...map[string]interface{}) *Tree {
	return &Tree{
		Name:  name,
		funcs: funcs,
	}
}

// ErrorContext returns a textual representation of the location of the node in the input text.
// The receiver is only used when the node does not have a pointer to the tree inside,
// which can occur in old code.
// func (t *Tree) ErrorContext(n ast.Node) (location, context string) {
// 	pos := int(n.Position())
// 	tree := n.tree()
// 	if tree == nil {
// 		tree = t
// 	}
// 	text := tree.text[:pos]
// 	byteNum := strings.LastIndex(text, "\n")
// 	if byteNum == -1 {
// 		byteNum = pos // On first line.
// 	} else {
// 		byteNum++ // After the newline.
// 		byteNum = pos - byteNum
// 	}
// 	lineNum := 1 + strings.Count(text, "\n")
// 	context = n.String()
// 	if len(context) > 20 {
// 		context = fmt.Sprintf("%.20s...", context)
// 	}
// 	return fmt.Sprintf("%s:%d:%d", tree.ParseName, lineNum, byteNum), context
// }

// errorf formats the error and terminates processing.
func (t *Tree) errorf(format string, args ...interface{}) {
	t.Root = nil
	format = fmt.Sprintf("%s: %s", t.ParseName, format)
	panic(fmt.Errorf(format, args...))
}

// error terminates processing.
func (t *Tree) error(err error) {
	t.errorf("%s", err)
}

// expect consumes the next token and guarantees it has the required type.
func (t *Tree) expect(expected token.TokenType, context string) token.Token {
	token := t.nextNonSpace()
	if token.Type() != expected {
		t.unexpected(token, context)
	}
	return token
}

// expectOneOf consumes the next token and guarantees it has one of the required types.
func (t *Tree) expectOneOf(expected1, expected2 token.TokenType, context string) token.Token {
	token := t.nextNonSpace()
	if token.Type() != expected1 && token.Type() != expected2 {
		t.unexpected(token, context)
	}
	return token
}

// unexpected complains about the token and terminates processing.
func (t *Tree) unexpected(tok token.Token, context string) {
	t.errorf("unexpected %s in %s", token.Tokens[tok.Type()], context)
}

// recover is the handler that turns panics into returns from the top level of Parse.
func (t *Tree) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		if t != nil {
			//t.lex.drain()
			t.stopParse()
		}
		*errp = e.(error)
	}
	return
}

// startParse initializes the parser, using the lexer.
func (t *Tree) startParse(funcs []map[string]interface{}, lex *lexer, treeSet map[string]*Tree) {
	t.Root = nil
	t.lex = lex
	t.vars = []string{"$"}
	t.funcs = funcs
	t.treeSet = treeSet
	t.topScope = ast.NewScope(nil)
}

// stopParse terminates parsing.
func (t *Tree) stopParse() {
	t.lex = nil
	t.vars = nil
	t.funcs = nil
	t.treeSet = nil
}

// Parse parses the template definition string to construct a representation of
// the template for execution. If either action delimiter string is empty, the
// default ("{{" or "}}") is used. Embedded template definitions are added to
// the treeSet map.
func (t *Tree) Parse(text, leftDelim, rightDelim string, treeSet map[string]*Tree, funcs ...map[string]interface{}) (tree *Tree, err error) {
	defer t.recover(&err)
	t.ParseName = t.Name
	t.startParse(funcs, lex(t.Name, text, leftDelim, rightDelim), treeSet)
	t.text = text
	t.parse()
	t.add()
	t.stopParse()
	return t, nil
}

// add adds tree to t.treeSet.
func (t *Tree) add() {
	tree := t.treeSet[t.Name]
	if tree == nil || IsEmptyTree(tree.Root) {
		t.treeSet[t.Name] = t
		return
	}
	if !IsEmptyTree(t.Root) {
		t.errorf("template: multiple definition of template %q", t.Name)
	}
}

// IsEmptyTree reports whether this tree (node) is empty of everything but space.
func IsEmptyTree(n ast.Node) bool {
	switch n := n.(type) {
	case nil:
		return true
	case *ast.ListNode:
		for _, node := range n.Nodes {
			if !IsEmptyTree(node) {
				return false
			}
		}
		return true
	default:
		panic("unknown node: " + n.String())
	}
	return false
}

// parse is the top-level parser for a template, essentially the same
// as itemList except it also parses {{define}} actions.
// It runs to EOF.
func (t *Tree) parse() (next ast.Node) {
	pTok := t.peek()
	t.Root = ast.NewList(pTok.Pos())

	for tok := t.peek(); tok.Type() != token.EOF; {
		n := t.parseStatement()
		if n == nil {
			fmt.Println("Parse ends")
			break
		}
		t.Root.Append(n)
	}
	return nil
}

func (t *Tree) parseStatement() ast.Node {
	fmt.Println("parse statement")
	const context = "statement"

	tok := t.nextNonSpace()

	switch tok.Type() {
	case token.EOF:
		fallthrough
	case token.ILLEGAL:
		return nil
	case token.COMMENT:
		return ast.NewCommentNode(tok.Pos(), tok.Val())
	case token.FN:
		return t.parseFunc(tok.Pos())
	case token.DATA:
		return t.parseDataStatement(tok.Pos())
	/*
		case token.LET:
			return t.parseLetExpr(tok.Pos())
	*/
	default:
		t.backup()
		return t.parseExpr()
	}

	return nil
}

func (t *Tree) parseLetExpr(pos int) ast.ExprNode {
	fmt.Println("parse let")
	const context = "let statement"

	var defns []*ast.DefnNode

	t.openScope()

	for {
		defnNode := t.parseDefn()
		defns = append(defns, defnNode)

		next := t.peekNonSpace()
		if next.Type() != token.COMMA {
			break
		}
		t.nextNonSpace()
	}

	t.expect(token.IN, context)

	letNode := ast.NewLetExpr(pos, defns, t.parseExpr())
	t.closeScope()
	return letNode
}

func (t *Tree) parseDefn() *ast.DefnNode {
	const context = "definition"

	iden := t.expect(token.IDENT, context)
	//if found nil then create new Object in scope else it is already in the scope
	if t.topScope.Lookup(iden.Val()) == nil {
		obj := ast.NewObj(iden.Val())
		t.topScope.Insert(obj)
	}

	t.expect(token.ASSIGN, context)
	exprNode := t.parseExpr()
	return ast.NewDefinition(iden.Pos(), iden.Val(), exprNode)
}

func (t *Tree) parseFunc(pos int) ast.ExprNode {
	const context = "function statement"
	name := t.expect(token.IDENT, context)

	/*
		scope := t.topScope

		for {
			if scope.Lookup(name.Val()) != nil {
				t.errorf("Function already declared in upper scope", context)
			}

			scope = t.topScope.Outer

			if scope == nil {
				break
			}
		}
	*/

	fmt.Println("Parsing function definition")
	obj := ast.NewObj(name.Val())
	t.topScope.Insert(obj)

	t.openScope()
	var params []string
	tok := t.nextNonSpace()

	switch tok.Type() {
	case token.LPAREN:
		next := t.peekNonSpace()
		if next.Type() != token.RPAREN {
			for {
				param := t.expect(token.IDENT, context)
				//if found nil then create new Object in scope else it is already in the scope
				if t.topScope.Lookup(param.Val()) == nil {
					obj := ast.NewObj(param.Val())
					t.topScope.Insert(obj)
				}
				fmt.Println(param.Val())
				params = append(params, param.Val())

				if next := t.peekNonSpace(); next.Type() != token.COMMA {
					break
				}
				t.nextNonSpace()
			}
			t.expect(token.RPAREN, context)
		} else {
			t.expect(token.RPAREN, context)
		}
	case token.IDENT:
		if t.topScope.Lookup(tok.Val()) == nil {
			obj := ast.NewObj(tok.Val())
			t.topScope.Insert(obj)
		}
		fmt.Println(tok.Val())
		params = append(params, tok.Val())
	case token.ARROW:
		body := t.parseExpr()
		tmpNode := ast.NewFunction(pos, name.Val(), params, body)
		t.closeScope()
		return tmpNode
	default:
		t.unexpected(tok, context)
	}

	t.expect(token.ARROW, context)
	body := t.parseExpr()
	tmpNode := ast.NewFunction(pos, name.Val(), params, body)
	t.closeScope()

	return tmpNode
}

func (t *Tree) parseFuncExpr(pos int) ast.ExprNode {
	const context = "function expression"
	t.openScope()
	tok := t.expectOneOf(token.LPAREN, token.IDENT, context)

	var params []string
	if tok.Type() == token.LPAREN {
		next := t.peekNonSpace()
		if next.Type() != token.RPAREN {
			for {
				param := t.expect(token.IDENT, context)
				//if found nil then create new Object in scope else it is already in the scope
				if t.topScope.Lookup(param.Val()) == nil {
					obj := ast.NewObj(param.Val())
					t.topScope.Insert(obj)
				}
				fmt.Println(param.Val())
				params = append(params, param.Val())

				if next := t.peekNonSpace(); next.Type() != token.COMMA {
					break
				}
				t.nextNonSpace()
			}
			t.expect(token.RPAREN, context)
		} else {
			t.expect(token.RPAREN, context)
		}

	} else {
		if t.topScope.Lookup(tok.Val()) == nil {
			obj := ast.NewObj(tok.Val())
			t.topScope.Insert(obj)
		}
		fmt.Println(tok.Val())
		params = append(params, tok.Val())
	}

	t.expect(token.ARROW, context)
	body := t.parseExpr()
	tmpNode := ast.NewFunctionExpression(pos, params, body)
	t.closeScope()
	return tmpNode
}

func (t *Tree) parseExpr() ast.ExprNode {
	const context = "expression"

	var retNode ast.ExprNode
	tok := t.nextNonSpace()

	switch tok.Type() {
	case token.ILLEGAL:
		t.errorf("%s", tok.Val())
	case token.BOOLEAN:
		boolean := ast.NewBool(tok.Pos(), tok.Val() == "True")
		retNode = boolean
	case token.STRING:
		s, err := strconv.Unquote(tok.Val())
		if err != nil {
			t.error(err)
		}
		retNode = ast.NewString(tok.Pos(), tok.Val(), s)
	case token.INT:
		number, err := ast.NewNumber(tok.Pos(), tok.Val(), tok.Type())
		if err != nil {
			t.error(err)
		}
		retNode = number
	case token.FLOAT:
		number, err := ast.NewNumber(tok.Pos(), tok.Val(), tok.Type())
		if err != nil {
			t.error(err)
		}
		retNode = number
	case token.IDENT:
		ident := t.useVar(tok.Pos(), tok.Val())
		retNode = ident
	case token.IF:
		retNode = t.parseIfStmt(tok.Pos())
	case token.LET:
		return t.parseLetExpr(tok.Pos())
	case token.FN:
		retNode = t.parseFuncExpr(tok.Pos())
	case token.LPAREN:
		retNode = t.parseExpr()
		t.expect(token.RPAREN, context)
	}

	if retNode == nil {
		t.backup()
		return nil
	}

	applicationNode := t.parseApplication(retNode)

	if applicationNode != nil {
		retNode = applicationNode
	}

	// Testing for infix expression
	infixNode := t.parseInfixExpr(retNode)

	if infixNode == nil {
		return retNode
	} else {
		return infixNode
	}
}

func (t *Tree) parseInfixExpr(left ast.ExprNode) ast.ExprNode {
	fmt.Println("trying to parse infix")
	const context = "infix expression"

	tok := t.nextNonSpace()
	fmt.Println(token.Tokens[tok.Type()])

	switch tok.Type() {
	// Binary operators
	case token.ADD:
		fallthrough
	case token.SUB:
		fallthrough
	case token.MUL:
		fallthrough
	case token.QUO:
		fallthrough
	case token.EQL:
		fallthrough
	case token.LSS:
		fallthrough
	case token.GTR:
		fallthrough
	case token.LAND:
		fallthrough
	case token.LOR:
		fallthrough
	case token.REM:
		right := t.parseExpr()
		if right == nil {
			t.errorf("expected expression after binary operator %s in %s", tok.Val(), context)
		}
		return ast.NewBinaryExpr(tok.Pos(), left, tok.Type(), right)
	}

	t.backup()
	return nil
}

func (t *Tree) parseApplication(left ast.ExprNode) ast.ExprNode {
	context := "function application"
	tok := t.nextNonSpace()

	var args []ast.ExprNode
	if tok.Type() == token.LPAREN {
		for {
			arg := t.parseExpr()
			if arg == nil {
				break
			}
			args = append(args, arg)
			if next := t.peekNonSpace(); next.Type() != token.COMMA {
				break
			}
			t.nextNonSpace()
		}
		t.expect(token.RPAREN, context)

		return ast.NewApplication(tok.Pos(), left, args)
	}

	t.backup()
	return nil
}

func (t *Tree) parseIfStmt(pos int) *ast.IfNode {
	const context = "If statement "
	condNode := t.parseExpr()
	t.expect(token.THEN, context)
	thenNode := t.parseExpr()
	tok := t.peek()
	if tok.Type() == token.ELSE {
		t.expect(token.ELSE, context)
		return ast.NewIfNode(pos, condNode, thenNode, t.parseExpr())
	}
	return ast.NewIfNode(pos, condNode, thenNode, t.parseExpr())
}

func (t *Tree) parseDataStatement(pos int) *ast.DataNode {
	const context = "type constructor definition"
	nameNode := t.expect(token.IDENT, context)

	var params []string
	var constrs []ast.ConstrNode

	next := t.peekNonSpace()
	if next.Type() == token.LSS {
		t.nextNonSpace()
		for {
			param := t.expect(token.IDENT, context)
			params = append(params, param.Val())

			if next := t.peekNonSpace(); next.Type() != token.COMMA {
				break
			}
			t.nextNonSpace()
		}
		t.expect(token.GTR, context)
	}

	t.expect(token.ASSIGN, context)

	next = t.peekNonSpace()
	for {
		constr := t.parseConstructor(params)
		constrs = append(constrs, *constr)
		obj := ast.NewObj(constr.Name)
		t.topScope.Insert(obj)

		if next := t.peekNonSpace(); next.Type() != token.OR {
			break
		}
		t.nextNonSpace()
	}

	return ast.NewDataNode(pos, nameNode.Val(), params, constrs)
}

func (t *Tree) parseConstructor(typeParams []string) *ast.ConstrNode {
	const context = "value constructor definition"

	var params []string

	nameNode := t.expect(token.IDENT, context)
	next := t.peekNonSpace()
	if next.Type() == token.LPAREN {
		t.nextNonSpace()
		for {
			param := t.expect(token.IDENT, context)
			// TODO: Check if param is present in typeParams
			params = append(params, param.Val())

			if next := t.peekNonSpace(); next.Type() != token.COMMA {
				break
			}
			t.nextNonSpace()
		}
		t.expect(token.RPAREN, context)
	}

	return ast.NewConstructorNode(nameNode.Pos(), nameNode.Val(), params)
}

func (t *Tree) parseTypeValue() {
	const context = "type value"

	t.expect(token.IDENT, context)

	next := t.peekNonSpace()
	if next.Type() == token.LSS {
	}
}

// hasFunction reports if a function name exists in the Tree's maps.
func (t *Tree) hasFunction(name string) bool {
	for _, funcMap := range t.funcs {
		if funcMap == nil {
			continue
		}
		if funcMap[name] != nil {
			return true
		}
	}
	return false
}

// popVars trims the variable list to the specified length
func (t *Tree) popVars(n int) {
	t.vars = t.vars[:n]
}

// useVar returns a node for a variable reference. It errors if the
// variable is not defined.
func (t *Tree) useVar(pos int, name string) ast.ExprNode {
	for s := t.topScope; s != nil; s = s.Outer {
		if obj := s.Lookup(name); obj != nil {
			v := ast.NewVariable(pos, name)
			// v.Obj = obj
			return v
		}
	}
	t.errorf("undefined variable %q", name)
	return nil
}

//Scoping Support
func (t *Tree) openScope() {
	t.topScope = ast.NewScope(t.topScope)
}
func (t *Tree) closeScope() {
	t.topScope = t.topScope.Outer
}
