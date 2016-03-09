package core
import (
	"fmt"
	"reflect"
	"testing"
)


type compileScTest struct {
	input ScDefn
	output GmCompiledSC
}

var compileScTests = []compileScTest{
	//{ScDefn{"Y", []Name{"f"}, ELet{true, [{"x", EAp{EVar("f"), EVar("x")}}], EVar("x")}}, GmCompiledSC{}},
// 	{ScDefn{"compose", []Name{"f","g","x"}, EAp{EVar("f"), EAp{EVar("g"), EVar("x")}}}, GmCompiledSC{"compose", 3, GmCode{Push(2), Push(1), Mkap{}, Push(0), Mkap{}, Slide(4), Unwind{}}}},
// 	{ScDefn{"twice", []Name{"f"}, EAp{EAp{EVar("compose"), EVar("f")}, EVar("f")}},GmCompiledSC{"twice", 1, GmCode{Push(0), Push(0), Pushglobal("compose"), Mkap{}, Mkap{} ,Slide(2), Unwind{}}}},    
// 	{ScDefn{"S", []Name{"f","g","x"}, EAp{ EAp {EVar("f"), EVar("x")},EAp {EVar("g"), EVar("x")}}}, GmCompiledSC{"S", 3, GmCode{Push(2), Push(1),Mkap{}, Push(2), Push(0), Mkap{}, Mkap{},
// Slide(4), Unwind{}}}},
	{ScDefn{
		"Let", []Name{"f"},
		ELet{true, []Defn{{Name("x"), EAp{EVar("f"), EVar("x")}}}, EVar("x")}}, GmCompiledSC{"Let", 1, GmCode{},
	}},
	{ScDefn{"K", []Name{"x", "y"}, EVar("x")}, GmCompiledSC{"K", 2, GmCode{Push(0), Slide(3), Unwind{}}}},
	{ScDefn{"K1", []Name{"x", "y"}, EVar("y")}, GmCompiledSC{"K1", 2, []Instruction{Push(1), Slide(3), Unwind{} }}},
	{ScDefn{"I", []Name{"x"}, EVar("x")}, GmCompiledSC{"I", 1, []Instruction{Push(0), Slide(2), Unwind{} }}},
	{ScDefn{"K2", []Name{"x","y"}, EVar("z")}, GmCompiledSC{"K2", 2, []Instruction{Pushglobal("z"), Slide(3), Unwind{}}}},
}

func printBody(body GmCode) {
	for _, inst := range body {
		fmt.Print(reflect.TypeOf(inst), inst, "  ")
	}
	fmt.Println()
}

func TestCompile(t *testing.T) {
	for _, cScTest := range compileScTests{
		result := compileSc(cScTest.input)
		if (result.Name == cScTest.input.Name) && (len(cScTest.input.Args)==result.Length) {
			// i := 0
			// for _,arr := range cScTest.output {
			// 	if arr != result.body[i] {
			// 		fmt.Println("Error while executing", cScTest.input.Name)
			// 		return
			// 	}
			// 	i = i + 1
			// }
			if cScTest.output.Name != result.Name {
				fmt.Println("Error while executing", cScTest.output.Name)
				return
			}

			if cScTest.output.Length != result.Length {
				fmt.Println("Number of Aguments Didn't Match", cScTest.output.Length)
				return
			}

			if len(cScTest.output.body) != len(result.body) {
				fmt.Println("Number of instructions didn't match for", cScTest.output.Name)
				fmt.Println("Got", len(result.body), "Expected", len(cScTest.output.body))
				printBody(result.body)
				return
			}

			for i, inst := range cScTest.output.body {
				if inst != result.body[i] {
					fmt.Println("Instruction does not match for", cScTest.input.Name)
					fmt.Println("Got", reflect.TypeOf(result.body[i]), result.body[i], "Expected", reflect.TypeOf(inst), inst)
					printBody(result.body)
					return
				}
			}

			fmt.Println(result.Name, "Done Successfully.")
		} else {
			fmt.Println("Error while executing", cScTest.input.Name)
		}
	}

	//fmt.Println(Compile(Program{ScDefn{"main", []Name{}, GmCode{ENum{true, false, false, 3, 3, 3, "3"}}}))

	fmt.Println(Compile(Program{
		ScDefn{"main", []Name{}, EAp{EVar("test"), ENum{true, false, false, 4, 4, 4, "4"}}},
		ScDefn{"test", []Name{"x"}, EVar("x")},
	}))

}