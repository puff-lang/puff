package parse

import (
	"flag"
	"fmt"
	"testing"
)

var debug = flag.Bool("debug", false, "show the errors produced by the main tests")

type parseTest struct {
	name string
	input string
	ok     bool
	result string // what the user would see in an error message.
}

const (
	noError  = true
	hasError = false
)

var parseTests = [] parseTest{
	{"empty", "", noError, ""},
	{"number", "0", noError, "0"},
	{"number", "-0", noError, ""},
	{"number", "73", noError, "73"},
	{"comment", "//This is comment \n  ", noError, ""},
	{"LetExpr","let a = 10, b = 5 in a + b", noError, "let a = 10, b = 5 in a + b"},
}

var builtins = map[string]interface{}{
	"printf": fmt.Sprintf,
}

func testParse(doCopy bool, t *testing.T) {
	textFormat := "%q"
	defer func() { textFormat = "%s" }()
	for _,test := range parseTests {
		tmpl, err := New(test.name).Parse(test.input, "", "", make(map[string]*Tree), builtins)
		switch {
		case err == nil && !test.ok:
			t.Errorf("%q: expected error; got none", test.name)
			continue
		case err != nil && test.ok:
			t.Errorf("%q: unexpected error: %v", test.name, err)
			continue
		case err != nil && !test.ok:
			// expected error, got one
			if *debug {
				fmt.Printf("%s: %s\n\t%s\n", test.name, test.input, err)
			}
			continue
		}
		var result string
		if doCopy {
			result = tmpl.Root.Copy().String()
		} else {
			result = tmpl.Root.String()
		}
		if result != test.result {
			t.Errorf("%s=(%q): got\n\t%v\nexpected\n\t%v", test.name, test.input, result, test.result)
		}
	}

}

func TestParse( t *testing.T) {
	testParse(false, t)
	fmt.Println("parsed")
}


// func collectTokens(src, left, right string) (tokenList []string) {
// 	l := lex("testing", src, left, right)

// 	for {
// 		tok := l.nextToken()
// 		tokenList = append(tokenList, token.Tokens[tok.Type()])
// 		if tok.Type() == token.EOF || tok.Type() == token.ILLEGAL {
// 			break
// 		}
// 	}
// 	return
// }

// func TestLetExpr(t *testing.T) {
// 	for _, test := range parseTests {
// 		fmt.Println(collectTokens(test.input, "", ""))
// 		tmpl, err := New("let expression test").Parse(test.input, "", "", make(map[string]*Tree), builtins)
// 		if err != nil {
// 			fmt.Println("!!! Something went wrong! !!!\n", err.Error())
// 		}
// 		fmt.Println("parsed", tmpl.Root.String())
// 	}
// }

