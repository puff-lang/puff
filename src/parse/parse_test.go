package parse

import (
	"flag"
	"fmt"
	"testing"
)

var debug = flag.Bool("debug", false, "show the errors produced by the main tests")

type parseTest struct {
	name   string
	input  string
	ok     bool
	result string // what the user would see in an error message.
}

const (
	noError  = true
	hasError = false
)

var parseTests = []parseTest{
	{"empty", "", noError, ""},
	{"number", "0", noError, "0"},
	{"number", "-0", noError, ""},
	{"number", "73", noError, "73"},
	{"comment", "//This is comment\n ", noError, ""},
	{"LetExpr", "let a = 10, b = 5 in a + b", noError, "let a = 10, b = 5 in a + b"},
	{"FunExprs", "fn (x, y, z) -> x + y + z", noError, "fn (x, y, z) -> x + y + z"},
	{"FunExprs", "fn x -> x + 1", noError, "fn (x) -> x + 1"},
	{"If Statement", "if 10 then 0 else 1", noError, "if 10 then 0 else 1"},
	{"Let with if", "let a = 2, b = 3 in if a then a + b else a -b", noError, "let a = 2, b = 3 in if a then a + b else a - b"},
	{"if without else", "if 10 then 3", noError, "if 10 then 3"},
	{"Nested if", "if 10 then if 2 then 3 else if 4 then 5", noError, "if 10 then if 2 then 3 else if 4 then 5"},
}

var builtins = map[string]interface{}{
	"printf": fmt.Sprintf,
}

func testParse(doCopy bool, t *testing.T) {
	textFormat := "%q"
	defer func() { textFormat = "%s" }()
	for _, test := range parseTests {
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

func TestParse(t *testing.T) {
	testParse(false, t)
	fmt.Println("parsed")
}
