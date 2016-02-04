package parse

import (
	"flag"
	"fmt"
	"testing"
)

var debug = flag.Bool("debug", false, "show the errors produced by the main tests")

type numberTest struct {
	text     string
	isInt    bool
	isFloat  bool
}


var numberTests = []numberTest{
	// basics
	{"0", true, true},
	{"-0", true, true}, // check that -0 is a uint.
	{"73", true, true},
}

var builtins = map[string]interface{}{
	"printf": fmt.Sprintf,
}

func collectTokens(src, left, right string) (tokenList []string) {
	l := lex("testing", src, left, right)

	for {
		token := l.nextToken()
		tokenList = append(tokenList, tokens[token.typ])
		if token.typ == EOF || token.typ == ILLEGAL {
			break
		}
	}
	return
}

func TestLetExpr(t *testing.T) {
	src := "fn x -> True"

	fmt.Println(collectTokens(src, "", ""))

	tmpl, err := New("let expression test").Parse(src, "", "", make(map[string]*Tree), builtins)

	if err != nil {
		t.Errorf("Something went wrong", "let parse", err)
	}
	fmt.Println(tmpl.Root.String());
}