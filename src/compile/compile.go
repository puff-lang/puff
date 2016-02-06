package compile

import (
	"fmt"
	"llvm.org/llvm/bindings/go/llvm"
	"parse"
)

var builtins = map[string]interface{}{
	"printf": fmt.Printf,
}

func Compile(src string, TheModule llvm.Module) llvm.Value {
	// tree, err := parse.Parse("Program", src, "", "", make(map[string]*parse.Tree), builtins)
	tree, err := parse.New("Program").Parse(src, "", "", make(map[string]*parse.Tree), builtins)

	// for node := range tree.Root.Nodes {
	// 	compileNode(node, TheModule)
	// }

	// return tree.Root.String();
	if err != nil {
		fmt.Println(err)
	}
	node := tree.Root.Nodes[0]
	return compileNode(node, TheModule)
}
