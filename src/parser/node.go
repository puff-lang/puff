package parse

import (
	"bytes"
	"strconv"
	"strings"
	"fmt"
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
	tree() *Tree
}

// NodeType identifies the type of a parse tree node.
type NodeType int

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
	NodeIdentifier  NodeType = iota // An identifier; always a function name.
	NodeBool                        // A boolean constant.
	NodeList                        // A list of Nodes.
	NodeNumber                      // A numerical constant.
	NodeString                      // A string constant.
	NodeVariable                    // A $ variable.

	NodeLet
	NodeDefn
	NodeExpr
	NodeFnExpr
)

// Nodes.

// ListNode holds a sequence of nodes.
type ListNode struct {
	NodeType
	Pos   int
	tr    *Tree
	Nodes []Node // The element nodes in lexical order.
}

func (t *Tree) newList(pos int) *ListNode {
	return &ListNode{tr: t, NodeType: NodeList, Pos: pos}
}

func (l *ListNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}

func (l *ListNode) tree() *Tree {
	return l.tr
}

func (l *ListNode) Position() int {
	return l.Pos;
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
	n := l.tr.newList(l.Pos)
	for _, elem := range l.Nodes {
		n.append(elem.Copy())
	}
	return n
}

func (l *ListNode) Copy() Node {
	return l.CopyList()
}

// StringNode holds a string constant. The value has been "unquoted".
type StringNode struct {
	NodeType
	Pos    int
	tr     *Tree
	Quoted string // The original text of the string, with quotes.
	Text   string // The string, after quote processing.
}

func (t *Tree) newString(pos int, orig, text string) *StringNode {
	return &StringNode{tr: t, NodeType: NodeString, Pos: pos, Quoted: orig, Text: text}
}

func (s *StringNode) String() string {
	return s.Quoted
}

func (s *StringNode) tree() *Tree {
	return s.tr
}

func (l *StringNode) Position() int {
	return l.Pos;
}

func (s *StringNode) Copy() Node {
	return s.tr.newString(s.Pos, s.Quoted, s.Text)
}

// BoolNode holds a boolean constant.
type BoolNode struct {
	NodeType
	Pos  int
	tr   *Tree
	True bool // The value of the boolean constant.
}

func (t *Tree) newBool(pos int, true bool) *BoolNode {
	return &BoolNode{tr: t, NodeType: NodeBool, Pos: pos, True: true}
}

func (b *BoolNode) String() string {
	if b.True {
		return "True"
	}
	return "False"
}

func (b *BoolNode) tree() *Tree {
	return b.tr
}

func (l *BoolNode) Position() int {
	return l.Pos;
}

func (b *BoolNode) Copy() Node {
	return b.tr.newBool(b.Pos, b.True)
}


// NumberNode holds a number: signed or unsigned integer, float, or complex.
// The value is parsed and stored under all the types that can represent the value.
// This simulates in a small amount of code the behavior of Go's ideal constants.
type NumberNode struct {
	NodeType
	Pos        int
	tr         *Tree
	IsInt      bool       // Number has an integral value.
	IsUint     bool       // Number has an unsigned integral value.
	IsFloat    bool       // Number has a floating-point value.
	Int64      int64      // The signed integer value.
	Uint64     uint64     // The unsigned integer value.
	Float64    float64    // The floating-point value.
	Text       string     // The original textual representation from the input.
}

func (t *Tree) newNumber(pos int, text string, typ tokenType) (*NumberNode, error) {
	n := &NumberNode{tr: t, NodeType: NodeNumber, Pos: pos, Text: text}
	switch typ {
	case CHAR:
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

func (n *NumberNode) String() string {
	return n.Text
}

func (n *NumberNode) tree() *Tree {
	return n.tr
}

func (l *NumberNode) Position() int {
	return l.Pos;
}

func (n *NumberNode) Copy() Node {
	nn := new(NumberNode)
	*nn = *n // Easy, fast, correct.
	return nn
}


// VariableNode holds a variable name.
type VariableNode struct {
	NodeType
	Pos   int
	tr    *Tree
	Ident string // Variable name.
}

func (t *Tree) newVariable(pos int, ident string) *VariableNode {
	return &VariableNode{tr: t, NodeType: NodeVariable, Pos: pos, Ident: ident}
}

func (v *VariableNode) String() string {
	return v.Ident
}

func (v *VariableNode) tree() *Tree {
	return v.tr
}

func (l *VariableNode) Position() int {
	return l.Pos;
}

func (v *VariableNode) Copy() Node {
	return &VariableNode{tr: v.tr, NodeType: NodeVariable, Pos: v.Pos, Ident: v.Ident}
}


// LetNode represents let defns in expr
type LetNode struct {
	NodeType
	Pos   int
	tr    *Tree
	Defns []*DefnNode
	Expr  *ExprNode
}

func (t *Tree) newLetExpr(pos int, defns []*DefnNode, exprNode *ExprNode) *LetNode {
	return &LetNode{tr: t, NodeType: NodeLet, Pos: pos, Defns: defns, Expr: exprNode}
}

func (v *LetNode) String() string {
	s := "let "
	for i, d := range v.Defns {
		s += d.String()
		if i > 0 && i < len(v.Defns) {
			s += ", "
		}
	}
	s += " in " + v.Expr.String()
	return s
}

func (v *LetNode) tree() *Tree {
	return v.tr
}

func (l *LetNode) Position() int {
	return l.Pos;
}

func (v *LetNode) Copy() Node {
	return &LetNode{tr: v.tr, NodeType: NodeLet, Pos: v.Pos, Defns: v.Defns, Expr: v.Expr}
}

// DefnNode represents a variable definition
type DefnNode struct {
	NodeType
	Pos   int
	tr    *Tree
	Var   string
	Expr  *ExprNode
}

func (t *Tree) newDefinition(pos int, variable string, exprNode *ExprNode) *DefnNode {
	return &DefnNode{tr: t, NodeType: NodeDefn, Pos: pos, Var: variable, Expr: exprNode}
}

func (v *DefnNode) String() string {
	return v.Var + " = " + v.Expr.String()
}

func (v *DefnNode) tree() *Tree {
	return v.tr
}

func (l *DefnNode) Position() int {
	return l.Pos;
}

func (v *DefnNode) Copy() Node {
	return &DefnNode{tr: v.tr, NodeType: NodeDefn, Pos: v.Pos, Var: v.Var, Expr: v.Expr}
}

// Abstract node to represent an expression
type ExprNode struct {
	NodeType
	Pos   int
	tr    *Tree
	node  Node
}

func (t *Tree) newExpression(node Node) *ExprNode {
	return &ExprNode{tr: t, NodeType: NodeExpr, Pos: node.Position(), node: node}
}

func (v *ExprNode) String() string {
	return v.node.String()
}

func (v *ExprNode) tree() *Tree {
	return v.tr
}

func (l *ExprNode) Position() int {
	return l.Pos;
}

func (v *ExprNode) Copy() Node {
	return &ExprNode{tr: v.tr, NodeType: NodeExpr, node: v.node}
}

type FnExprNode struct {
	NodeType
	Pos    int
	tr     *Tree
	Params []string
	Body   *ExprNode
}

func (t *Tree) newFunctionExpression(pos int, params []string, body *ExprNode) *FnExprNode {
	return &FnExprNode{tr: t, NodeType: NodeFnExpr, Params: params, Body: body}
}

func (v *FnExprNode) String() string {
	s := "fn ("
	for i, d := range v.Params {
		s += d
		if i > 0 && i < len(v.Params) {
			s += ", "
		}
	}
	s += ") -> "
	s += v.Body.String()
	return s
}

func (v *FnExprNode) tree() *Tree {
	return v.tr
}

func (l *FnExprNode) Position() int {
	return l.Pos;
}

func (v *FnExprNode) Copy() Node {
	return &FnExprNode{tr: v.tr, NodeType: NodeFnExpr, Params: v.Params, Body: v.Body}
}
