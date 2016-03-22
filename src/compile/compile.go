package compile	
import (
	"fmt"
	"core"
	// "llvm.org/llvm/bindings/go/llvm"
	"parse"
)

var builtins = map[string]interface{}{
	"printf": fmt.Printf,
}

// func Compile(src string, TheModule llvm.Module) llvm.Value {
// 	// tree, err := parse.Parse("Program", src, "", "", make(map[string]*parse.Tree), builtins)
// 	tree, err := parse.New("Program").Parse(src, "", "", make(map[string]*parse.Tree), builtins)

// 	// for node := range tree.Root.Nodes {
// 	// 	compileNode(node, TheModule)
// 	// }

// 	// return tree.Root.String();
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	node := tree.Root.Nodes[0]
// 	return compileNode(node, TheModule)
// }


func Translate(src string) core.CoreExpr {
	tree, err := parse.New("Program").Parse(src, "", "", make(map[string]*parse.Tree), builtins)

	if err != nil {
		fmt.Println(err)
	}
	node := tree.Root.Nodes[0]
	return translateNode(node)
}

func run(str string) string { //Almost Done
	contentProgram := core.Compile(core.Program{
            core.ScDefn{"main", []core.Name{}, Translate(string(str))},
    	})
	result := core.EvalState(contentProgram) // []GmState
	fmt.Println(result)
	return string("result")
}


