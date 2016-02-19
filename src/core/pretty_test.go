package core

import (
	"fmt"
	"testing"
)

func TestPretty(t *testing.T) {
	src := EAp{EVar("f"), EVar("x")}

	formatted := PpExpr(src)

	if formatted != "f(x)" {
		t.Errorf("expected x got %s", formatted)
	}
	fmt.Println(formatted)
}