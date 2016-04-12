package ast

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"token"
)

// A Node is an element in the parse tree. The interface is trivial.
// The interface contains an unexported method so that only
// types local to this package can satisfy it.
type Node interface {
	Type() NodeType
	String() string
	// Copy does a deep copy of the Node and all its components.
	// To avoid type assertions, some XxxNodes also have specialized
	// CopyXxx methods that return *XxxNode.
	Copy() Node
	Position() int // byte position of start of node in full original input string
	// tree returns the containing *Tree.
	// It is unexported so all implementations of Node are in this package.
	// tree() *Tree
}

// NodeType identifies the type of a parse tree node.
type NodeType int

type ExprNode interface {
	Node
	exprNode()
}

/*
type Pos int

func (p Pos) Position() int {
	return p
}
*/

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeIdentifier NodeType = iota // An identifier; always a function name.
	NodeBool                       // A boolean constant.
	NodeList                       // A list of Nodes.
	NodeNumber                     // A numerical constant.
	NodeString                     // A string constant.
	NodeVariable                   // A $ variable.

	NodeLet
	NodeDefn
	NodeExpr
	NodeFnExpr
	NodeFn
	NodeAp
	NodeBinaryExpr
	NodeIf
	NodeComment
)

// Nodes.

// ListNode holds a sequence of nodes.
type ListNode struct {
	NodeType
	Pos int
	// tr    *Tree
	Nodes []Node // The element nodes in lexical order.
}

func NewList(pos int) *ListNode {
	return &ListNode{NodeType: NodeList, Pos: pos}
}

func (l *ListNode) Append(n Node) {
	l.Nodes = append(l.Nodes, n)
}

// func (l *ListNode) tree() *Tree {
// 	return l.tr
// }

func (l *ListNode) Position() int {
	return l.Pos
}

func (l *ListNode) String() string {
	b := new(bytes.Buffer)
	for _, n := range l.Nodes {
		fmt.Fprint(b, n)
	}
	return b.String()
}

func (l *ListNode) CopyList() *ListNode {
	if l == nil {
		return l
	}
	n := NewList(l.Pos)
	for _, elem := range l.Nodes {
		n.Append(elem.Copy())
	}
	return n
}

func (l *ListNode) Copy() Node {
	return l.CopyList()
}

// StringNode holds a string constant. The value has been "unquoted".
type StringNode struct {
	NodeType
	Pos int
	// tr     *Tree
	Quoted string // The original text of the string, with quotes.
	Text   string // The string, after quote processing.
}

func NewString(pos int, orig, text string) *StringNode {
	return &StringNode{NodeType: NodeString, Pos: pos, Quoted: orig, Text: text}
}

func (s *StringNode) String() string {
	return s.Quoted
}

// func (s *StringNode) tree() *Tree {
// 	return s.tr
// }

func (l *StringNode) Position() int {
	return l.Pos
}

func (s *StringNode) Copy() Node {
	return NewString(s.Pos, s.Quoted, s.Text)
}

func (*StringNode) exprNode() {}

// BoolNode holds a boolean constant.
type BoolNode struct {
	NodeType
	Pos int
	// tr   *Tree
	True bool // The value of the boolean constant.
}

func NewBool(pos int, true bool) *BoolNode {
	return &BoolNode{NodeType: NodeBool, Pos: pos, True: true}
}

func (b *BoolNode) String() string {
	if b.True {
		return "True"
	}
	return "False"
}

// func (b *BoolNode) tree() *Tree {
// 	return b.tr
// }

func (l *BoolNode) Position() int {
	return l.Pos
}

func (b *BoolNode) Copy() Node {
	return NewBool(b.Pos, b.True)
}

func (*BoolNode) exprNode() {}

// NumberNode holds a number: signed or unsigned integer, float, or complex.
// The value is parsed and stored under all the types that can represent the value.
// This simulates in a small amount of code the behavior of Go's ideal constants.
type NumberNode struct {
	NodeType
	Pos int
	// tr         *Tree
	IsInt   bool    // Number has an integral value.
	IsUint  bool    // Number has an unsigned integral value.
	IsFloat bool    // Number has a floating-point value.
	Int64   int64   // The signed integer value.
	Uint64  uint64  // The unsigned integer value.
	Float64 float64 // The floating-point value.
	Text    string  // The original textual representation from the input.
}

func NewNumber(pos int, text string, typ token.TokenType) (*NumberNode, error) {
	n := &NumberNode{NodeType: NodeNumber, Pos: pos, Text: text}
	switch typ {
	case token.CHAR:
		rune, _, tail, err := strconv.UnquoteChar(text[1:], text[0])
		if err != nil {
			return nil, err
		}
		if tail != "'" {
			return nil, fmt.Errorf("malformed character constant: %s", text)
		}
		n.Int64 = int64(rune)
		n.IsInt = true
		n.Uint64 = uint64(rune)
		n.IsUint = true
		n.Float64 = float64(rune) // odd but those are the rules.
		n.IsFloat = true
		return n, nil
	}
	// Do integer test first so we get 0x123 etc.
	u, err := strconv.ParseUint(text, 0, 64) // will fail for -0; fixed below.
	if err == nil {
		n.IsUint = true
		n.Uint64 = u
	}
	i, err := strconv.ParseInt(text, 0, 64)
	if err == nil {
		n.IsInt = true
		n.Int64 = i
		if i == 0 {
			n.IsUint = true // in case of -0.
			n.Uint64 = u
		}
	}
	// If an integer extraction succeeded, promote the float.
	if n.IsInt {
		n.IsFloat = true
		n.Float64 = float64(n.Int64)
	} else if n.IsUint {
		n.IsFloat = true
		n.Float64 = float64(n.Uint64)
	} else {
		f, err := strconv.ParseFloat(text, 64)
		if err == nil {
			// If we parsed it as a float but it looks like an integer,
			// it's a huge number too large to fit in an int. Reject it.
			if !strings.ContainsAny(text, ".eE") {
				return nil, fmt.Errorf("integer overflow: %q", text)
			}
			n.IsFloat = true
			n.Float64 = f
			// If a floating-point extraction succeeded, extract the int if needed.
			if !n.IsInt && float64(int64(f)) == f {
				n.IsInt = true
				n.Int64 = int64(f)
			}
			if !n.IsUint && float64(uint64(f)) == f {
				n.IsUint = true
				n.Uint64 = uint64(f)
			}
		}
	}
	if !n.IsInt && !n.IsUint && !n.IsFloat {
		return nil, fmt.Errorf("illegal number syntax: %q", text)
	}
	return n, nil
}

func (n NumberNode) String() string {
	return n.Text
}

// func (n *NumberNode) tree() *Tree {
// 	return n.tr
// }

func (l NumberNode) Position() int {
	return l.Pos
}

func (n NumberNode) Copy() Node {
	nn := new(NumberNode)
	*nn = n // Easy, fast, correct.
	return nn
}

func (*NumberNode) exprNode() {}

// VariableNode holds a variable name.
type VariableNode struct {
	NodeType
	Pos int
	// tr    *Tree
	Ident string // Variable name.
	// Value *Node
}

func NewVariable(pos int, ident string) *VariableNode {
	return &VariableNode{NodeType: NodeVariable, Pos: pos, Ident: ident}
}

func (v *VariableNode) String() string {
	return v.Ident
}

// func (v *VariableNode) tree() *Tree {
// 	return v.tr
// }

func (l *VariableNode) Position() int {
	return l.Pos
}

func (v *VariableNode) Copy() Node {
	return &VariableNode{NodeType: NodeVariable, Pos: v.Pos, Ident: v.Ident}
}


func (v *VariableNode) exprNode() {}



// LetNode represents let defns in expr
type LetNode struct {
	NodeType
	Pos int
	// tr    *Tree
	Defns []*DefnNode
	Expr  ExprNode
}

func NewLetExpr(pos int, defns []*DefnNode, exprNode ExprNode) *LetNode {
	return &LetNode{NodeType: NodeLet, Pos: pos, Defns: defns, Expr: exprNode}
}

func (v *LetNode) String() string {
	s := "let "
	for i, d := range v.Defns {
		s += d.String()
		if i >= 0 && i < len(v.Defns) - 1 {
			s += ", "
		}
	}
	s += " in " + v.Expr.String()
	return s
}

// func (v *LetNode) tree() *Tree {
// 	return v.tr
// }

func (l *LetNode) Position() int {
	return l.Pos
}

func (v *LetNode) Copy() Node {
	return &LetNode{NodeType: NodeLet, Pos: v.Pos, Defns: v.Defns, Expr: v.Expr}
}

func (*LetNode) exprNode() {}

// DefnNode represents a variable definition
type DefnNode struct {
	NodeType
	Pos int
	// tr    *Tree
	Var  string
	Expr ExprNode
}

func (*DefnNode) exprNode() {}

func NewDefinition(pos int, variable string, exprNode ExprNode) *DefnNode {
	return &DefnNode{NodeType: NodeDefn, Pos: pos, Var: variable, Expr: exprNode}
}

func (v *DefnNode) String() string {
	return v.Var + " = " + v.Expr.String()
}

// func (v *DefnNode) tree() *Tree {
// 	return v.tr
// }

func (l *DefnNode) Position() int {
	return l.Pos
}

func (v *DefnNode) Copy() Node {
	return &DefnNode{NodeType: NodeDefn, Pos: v.Pos, Var: v.Var, Expr: v.Expr}
}


type FnExprNode struct {
	NodeType
	Pos int
	// tr     *Tree
	Params []string
	Body   ExprNode
}

func NewFunctionExpression(pos int, params []string, body ExprNode) *FnExprNode {
	return &FnExprNode{NodeType: NodeFnExpr, Params: params, Body: body}
}

func (v *FnExprNode) String() string {
	s := "fn ("
	for i, d := range v.Params {
		s += d
		if i >= 0 && i < len(v.Params) -1 {
			s += ", "
		}
	}
	s += ") -> "
	s += v.Body.String()
	return s
}

// func (v *FnExprNode) tree() *Tree {
// 	return v.tr
// }

func (l *FnExprNode) Position() int {
	return l.Pos
}

func (v *FnExprNode) Copy() Node {
	return &FnExprNode{NodeType: NodeFnExpr, Params: v.Params, Body: v.Body}
}

func (*FnExprNode) exprNode() {}

type FnNode struct {
	NodeType
	Pos int
	// tr     *Tree
	Name   string
	Params []string
	Body   ExprNode
}

func NewFunction(pos int, name string, params []string, body ExprNode) *FnNode {
	return &FnNode{NodeType: NodeFn, Name: name, Params: params, Body: body}
}

func (v *FnNode) String() string {
	s := "fn ("
	for i, d := range v.Params {
		s += d
		if i >= 0 && i < len(v.Params) -1 {
			s += ", "
		}
	}
	s += ") -> "
	s += v.Body.String()
	return s
}

// func (v *FnExprNode) tree() *Tree {
// 	return v.tr
// }

func (l *FnNode) Position() int {
	return l.Pos
}

func (v *FnNode) Copy() Node {
	return &FnNode{NodeType: NodeFn, Name: v.Name, Params: v.Params, Body: v.Body}
}

func (*FnNode) exprNode() {}

type ApNode struct {
	NodeType
	Pos int
	// tr     *Tree
	Left  ExprNode
	Args  []ExprNode
}

func NewApplication(pos int, left ExprNode, args []ExprNode) *ApNode {
	return &ApNode{NodeType: NodeAp, Left: left, Args: args}
}

func (v *ApNode) String() string {
	fmt.Println("No. of args: ", len(v.Args))
	s := v.Left.String() + "("
	for i, d := range v.Args {
		s += d.String()
		if i >= 0 && i < len(v.Args) - 1 {
			s += ", "
		}
	}
	s += ")"
	return s
}

func (l *ApNode) Position() int {
	return l.Pos
}

func (v *ApNode) Copy() Node {
	return &ApNode{NodeType: NodeAp, Left: v.Left, Args: v.Args}
}

func (*ApNode) exprNode() {}

// Infix Binary expression
type BinaryExprNode struct {
	NodeType
	Pos int
	// tr     *Tree
	Left    ExprNode
	Right   ExprNode
	Op      token.TokenType
}

func NewBinaryExpr(pos int, left ExprNode, tokenType token.TokenType, right ExprNode) *BinaryExprNode {
	return &BinaryExprNode{NodeType: NodeBinaryExpr, Pos: pos, Left: left, Right: right, Op: tokenType}
}

func (v *BinaryExprNode) String() string {
	return v.Left.String() + " " + token.Tokens[v.Op] + " " + v.Right.String()
}

func (l *BinaryExprNode) Position() int {
	return l.Pos
}

func (v *BinaryExprNode) Copy() Node {
	return &BinaryExprNode{NodeType: NodeBinaryExpr, Pos: v.Pos, Left: v.Left, Right: v.Right, Op: v.Op}
}

func (*BinaryExprNode) exprNode() {}



type IfNode struct{
	NodeType
	Pos int
	Cond ExprNode
	Then ExprNode
	Else ExprNode
}
func (v *IfNode) exprNode() {}

func NewIfNode(pos int, condStmt ExprNode, thenStmt ExprNode, elseStmt ExprNode) *IfNode{
	return &IfNode{NodeType: NodeIf, Cond: condStmt, Then: thenStmt, Else: elseStmt}
}

func (v *IfNode) String() string {
	s:= "if " + v.Cond.String() + " then " + v.Then.String()
	if v.Else != nil {
		s += " else " + v.Else.String()   	
	}  
	return s 
}

func  (v *IfNode) Position() int  {
	return v.Pos
}

func (v *IfNode) Copy() Node {
	return &IfNode{NodeType: NodeIf, Cond: v.Cond, Then: v.Then, Else: v.Then}
}




type CommentNode struct {
	NodeType
	Pos int
	Text string
}
func NewCommentNode(pos int, text string ) *CommentNode {
	return &CommentNode{NodeType: NodeComment, Pos: pos, Text: text}	
}
func (c *CommentNode) Copy() Node {
	return &CommentNode{NodeType: NodeComment, Pos: c.Pos, Text: c.Text}
}
func (c *CommentNode) Position() int {
	 return c.Pos
}
func (c *CommentNode) End() int {
	 return (int(c.Position()) + len(c.Text)) 
}
func (c *CommentNode) String() string {
	// text := c.Text
	// if strings.Contains(text, "\n"){
	// 	text = "Block COMMENT:" + text
	// } else {
	// 	text = "Line COMMENT:" +text
	// }
	// return text 
	return ""
}

// A Scope maintains the set of named language entities declared
// in the scope and a link to the immediately surrounding (outer)
// scope.
type Scope struct {
	Outer *Scope
	Objects map[string]*Object
}
// NewScope creates a new scope nested in the outer scope.
func NewScope(outer *Scope) *Scope {
	const num = 5 //Initial Scope Capacity
	return &Scope{Outer: outer, Objects: make(map[string]*Object, num)}
}
// Lookup returns the object with the given name if it is found 
//in scope s, otherwise it returns nil. Outer scopes are ignored
func (s *Scope) Lookup(name string) *Object {
	return s.Objects[name]
}
// Insert attempts to insert a named object obj into the scope s.
// If the scope already contains an object alt with the same name,
// Insert leaves the scope unchanged and returns alt. Otherwise
// it inserts obj and returns nil.
func (s *Scope) Insert(obj *Object) (alt *Object) {
	if alt = s.Objects[obj.Name]; alt == nil {
		s.Objects[obj.Name] = obj
	}
	return
}
func (s *Scope) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "scope %p {", s)
	if s != nil && len(s.Objects) > 0 {
		fmt.Fprintln(&buf)
		for _, obj := range s.Objects {
			fmt.Fprintf(&buf, "\t %s\n", obj.Name)
		}
	}
	fmt.Fprintf(&buf, "}\n")
	return buf.String()
}

// Objects
// An Object describes a named language entity such as a package,
type Object struct{
	//Kind token.IDENT
	Name string
	//Value *NumberNode
}

func NewObj( name string ) *Object {
	// return &Object{Kind: kind, Name: name, Value: numNode}
	return &Object{Name: name}
}


















