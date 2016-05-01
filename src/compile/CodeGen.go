package compile

import (
	"bytes"
	"core"
	"fmt"
	"text/template"
	// "reflect"
	"io/ioutil"
	// "os"
	// "bufio"
)

type Reg int

type LLVMIR string

type LLVMValue interface {
	isShow()
}

type LLVMNum int

func (e LLVMNum) isShow() {}

type LLVMReg Reg

func (e LLVMReg) isShow() {}

type LLVMStackAddr int

func (e LLVMStackAddr) isShow() {}

type LLVMStack []LLVMValue

type Arity int
type Obj struct {
	arity Arity
	gmc   core.GmCode
}
type NameArityCodeMapping struct {
	name core.Name
	obj  Obj
}

var funPrefix string = "_"

var numTag int = 1

var globalTag int = 2

var apTag int = 3

var initialReg Reg = Reg(1)

var initialInstructionNum int = 1

func nextReg(reg Reg) Reg {
	return reg + 1
}

var templatesPath string = "src/compile/llvmTemplates/"

var codegenPath string = "../compile/CodeGen/"

func check(e error) { //Done
	if e != nil {
		panic(e)
	}
}

func SaveLLVMIR(ir LLVMIR) { //Done
	var filePath string = "./out.ll"
	d1 := []byte(ir)
	err := ioutil.WriteFile(filePath, d1, 0644)
	check(err)
}

func gettemplates() [26]string { //Done
	files, _ := ioutil.ReadDir(templatesPath)
	var templates [26]string
	for i, f := range files {
		templates[i] = f.Name()[:len(f.Name())-3]
	}
	return templates
}

func getStringTemplate(nm string, templates [26]string) string { //Done
	for _, temp := range templates {
		if nm == temp {
			b, err := ioutil.ReadFile(templatesPath + temp + ".st")
			if err != nil {
				panic(err)
			}
			return string(b)
		}
	}
	return ""
}

func GenLLVMIR(program core.GmState) LLVMIR { //Done
	templates := gettemplates()
	fmt.Println(program)
	return genProgramLLVMIR(templates, program)
}

type ProgramAttr struct { //Done
	Scs        string
	ConstrFuns string
}

func setProgramAttrib(b string, inv ProgramAttr) string { //Done
	tmpl, err := template.New("test").Parse(string(b))
	if err != nil {
		panic(err)
	}
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, inv)
	return doc.String()
}

func genProgramLLVMIR(templates [26]string, gmc core.GmState) LLVMIR { //Done ( p core.Program/)
	state := gmc
	globals := core.GetGlobals(state)
	heap := core.GetHeap(state)
	// fmt.Println("Globals: ",globals)
	// fmt.Println("Heaps: ",heap)
	temp := getStringTemplate("program", templates)
	mapping := createNameArityCodeMapping(heap, globals) //Problem Comes Here
	fmt.Println("mapping: ", mapping)
	scsTemplates := genScsLLVMIR(mapping, templates, globals)
	// constrs :=
	// constrsDash := createConstr(templates,LLVMIR(""), constrs)
	constrsDash := ""
	return LLVMIR(setProgramAttrib(temp, ProgramAttr{string(scsTemplates), constrsDash}))
}

func createConstr(templates [26]string, llir LLVMIR, name string, tag int, arity int) LLVMIR { //Done
	packTmpl := getStringTemplate("pack", templates)
	packTmplDash := packAttrib(packTmpl, PackStruct{0, tag, arity})

	updatTmpl := getStringTemplate("update", templates)
	updatTmplDash := setManyAttrib(updatTmpl, Inventory{core.Push(0), 1})

	constrTmpl := getStringTemplate("constr", templates)
	constrTmplDash := setConstrAttrib(constrTmpl, ConstrAttrib{tag, arity, updatTmplDash, packTmplDash})

	return LLVMIR(constrTmplDash) + llir
}

func genScsLLVMIR(mapping []NameArityCodeMapping, templates [26]string, gmg core.GmGlobals) LLVMIR { //Done
	temp := getStringTemplate("sc", templates)
	tmp := LLVMIR("")
	for _, obj := range gmg {
		tmp = tmp + mapScDefn(mapping, LLVMIR(temp), templates, obj)
	}
	return tmp
}

//Done
func mapScDefn(mapping []NameArityCodeMapping, temp LLVMIR, templates [26]string, gmg core.Object) LLVMIR {
	for _, nacm := range mapping {
		if gmg.Name == nacm.name {
			body := genScLLVMIR(nacm, templates, nacm.obj.gmc)
			name := mkFunName(string(nacm.name))
			return LLVMIR(setScAttrib(string(temp), ScsAttrib{string(body), string(name)}))
		}
	}
	return LLVMIR("")
}

func createEntry(gmh core.GmHeap, name core.Name, addr core.Addr) NameArityCodeMapping { //Done
	node := gmh.HLookup(addr)
	fmt.Println(name, " : ", node)
	switch node.(type) {
	case core.NGlobal:
		nd := node.(core.NGlobal)
		return NameArityCodeMapping{name, Obj{Arity(nd.Nargs), nd.GmC}}
	default:
		return NameArityCodeMapping{}
	}
}

func createNameArityCodeMapping(gmh core.GmHeap, gmg core.GmGlobals) []NameArityCodeMapping {
	globalMapping := []NameArityCodeMapping{}
	for _, obj := range gmg {
		globalMapping = append(globalMapping, createEntry(gmh, obj.Name, obj.Addr))
	}
	globalMapping = append(globalMapping, NameArityCodeMapping{core.Name("connet"), Obj{Arity(0), core.GmCode{}}})
	globalMapping = append(globalMapping, NameArityCodeMapping{core.Name("send"), Obj{Arity(1), core.GmCode{}}})
	return globalMapping
}

func genScLLVMIR(mapping NameArityCodeMapping, templates [26]string, gmc core.GmCode) string {
	state := UseIR{initialReg, LLVMStack{}, LLVMIR(""), initialInstructionNum}
	for _, instr := range gmc {
		state = translateToLLVMIR(mapping, templates, state, instr)
		// fmt.Println("Instruction Number: ",state.ninstr)
	}
	return string(state.ir)
}

type UseIR struct { //Done
	reg    Reg
	stack  LLVMStack
	ir     LLVMIR
	ninstr int
}

//Done
func translateToLLVMIR(mapping NameArityCodeMapping, templates [26]string, useir UseIR, instr core.Instruction) UseIR {
	switch instr.(type) {
	case core.Update:
		fmt.Println("update: ", useir.ninstr)
		temp := getStringTemplate("update", templates)
		templateDash := setManyAttrib(temp, Inventory{instr, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Push:
		fmt.Println("push: ", useir.ninstr)
		temp := getStringTemplate("push", templates)
		templateDash := setManyAttrib(temp, Inventory{instr, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Pop:
		fmt.Println("pop: ", useir.ninstr)
		temp := getStringTemplate("pop", templates)
		templateDash := setManyAttrib(temp, Inventory{instr, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Pushint:
		fmt.Println("pushint: ", useir.ninstr)
		temp := getStringTemplate("pushint", templates)
		templateDash := setManyAttrib(temp, Inventory{instr, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Pushglobal:
		fmt.Println("pushglobal: ", useir.ninstr)
		temp := getStringTemplate("pushglobal", templates)
		templateDash := setPushGlobalAttrib(temp, GlobalAttr{1, instr, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Mkap:
		fmt.Println("mkap: ", useir.ninstr)
		temp := getStringTemplate("mkap", templates)
		templateDash := setManyAttrib(temp, Inventory{core.Unwind{}, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Unwind:
		fmt.Println("unwind: ", useir.ninstr)
		temp := getStringTemplate("unwind", templates)
		templateDash := setManyAttrib(temp, Inventory{core.Unwind{}, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Eval:
		fmt.Println("eval: ", useir.ninstr)
		temp := getStringTemplate("eval", templates)
		templateDash := setManyAttrib(temp, Inventory{core.Unwind{}, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Pushbasic:
		fmt.Println("Pushbasic: ", useir.ninstr)
		temp := getStringTemplate("pushbasic", templates)
		templateDash := setManyAttrib(temp, Inventory{instr, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.MkInt:
		fmt.Println("MkInt: ", useir.ninstr)
		temp := getStringTemplate("mkint", templates)
		templateDash := setManyAttrib(temp, Inventory{core.Unwind{}, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Get:
		fmt.Println("get: ", useir.ninstr)
		temp := getStringTemplate("get", templates)
		templateDash := setManyAttrib(temp, Inventory{core.Unwind{}, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Alloc:
		fmt.Println("alloc: ", useir.ninstr)
		temp := getStringTemplate("alloc", templates)
		templateDash := setManyAttrib(temp, Inventory{instr, useir.ninstr})
		return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

	case core.Add:
		fmt.Println("add: ", useir.ninstr)
		llir := mkArithTmpl(templates, "add", useir.ninstr)
		return translateBinOp(templates, useir, llir)

	case core.Sub:
		fmt.Println("sub: ", useir.ninstr)
		llir := mkArithTmpl(templates, "sub", useir.ninstr)
		return translateBinOp(templates, useir, llir)

	case core.Mul:
		fmt.Println("mul: ", useir.ninstr)
		llir := mkArithTmpl(templates, "mul", useir.ninstr)
		return translateBinOp(templates, useir, llir)

	case core.Div:
		fmt.Println("div: ", useir.ninstr)
		llir := mkArithTmpl(templates, "udiv", useir.ninstr)
		return translateBinOp(templates, useir, llir)

	case core.Mod:
		fmt.Println("mod: ", useir.ninstr)
		llir := mkArithTmpl(templates, "urem", useir.ninstr)
		return translateBinOp(templates, useir, llir)

	case core.Eq:
		fmt.Println("eq", useir.ninstr)
		llir := mkRelationalTmpl(templates, "eq", useir.ninstr)
		return translateBinOp(templates, useir, llir)

	case core.Ne:
		fmt.Println("ne", useir.ninstr)
		llir := mkRelationalTmpl(templates, "ne", useir.ninstr)
		return translateBinOp(templates, useir, llir)

	case core.Lt:
		fmt.Println("lt", useir.ninstr)
		llir := mkRelationalTmpl(templates, "ult", useir.ninstr)
		return translateBinOp(templates, useir, llir)

	case core.Le:
		fmt.Println("le", useir.ninstr)
		llir := mkRelationalTmpl(templates, "ule", useir.ninstr)
		return translateBinOp(templates, useir, llir)

	case core.Gt:
		fmt.Println("gt", useir.ninstr)
		llir := mkRelationalTmpl(templates, "uge", useir.ninstr)
		return translateBinOp(templates, useir, llir)

	case core.Ge:
		fmt.Println("ge", useir.ninstr)
		llir := mkRelationalTmpl(templates, "uge", useir.ninstr)
		return translateBinOp(templates, useir, llir)

		// case core.Pack:
		// 	fmt.Println("Pack: ", useir.ninstr)
		// 	temp := getStringTemplate("pack", templates)
		// 	templateDash := setManyAttrib(temp, Inventory{instr, useir.ninstr})
		// 	return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), useir.ninstr + 1}

		// case core.CasejumpSimple:
		// 	fmt.Println("CasejumpSimple: ", useir.ninstr)
		// 	temp := getStringTemplate("casejumpsimple", templates)
		// 	templateDash, ninstrDash := translateCase(mapping, templates, instr.(type).)
		// 	return UseIR{useir.reg, useir.stack, useir.ir + LLVMIR(templateDash), ninstrDash}

	}
	return UseIR{}
}

func translateBinOp(templates [26]string, useir UseIR, llir LLVMIR) UseIR { //Done
	return UseIR{useir.reg, useir.stack, useir.ir + llir, (useir.ninstr + 1)}
}

func mkArithTmpl(templates [26]string, instr string, ninstr int) LLVMIR { //Done
	llvmName := instr
	temp := getStringTemplate("arith", templates)
	return LLVMIR(setArithAttrib(temp, Arithmatic{ninstr, llvmName}))
}

func mkRelationalTmpl(templates [26]string, instr string, ninstr int) LLVMIR { // Remaining: with true & false Tags
	llvmName := instr
	temp := getStringTemplate("relational", templates)
	return LLVMIR(setRelationalAttrib(temp, Relational{ninstr, llvmName, 1, 0}))
}

func mkFunName(name string) string { //Done
	Names := map[string]string{
		"+":  "add",
		"-":  "sub",
		"*":  "mul",
		"/":  "udiv",
		"%":  "mod",
		">":  "gt",
		"<":  "lt",
		"==": "eql",
	}

	value, ok := Names[name]

	if ok {
		return funPrefix + value
	} else {
		return funPrefix + name
	}
}

// func translateCase() (LLVMIR, int){

// }

// func translateBranch() (LLVMIR) {

// }

// func translateAlts() (LLVMIR) {

// }

// func translateAlt() () {

// }

// type Casejump struct {
// 	Ninstr int
// 	Branches
// 	Alts
// }
// func setCaseJumpAttrib(temp string, casejump Casejump) string, int{

// }

type ConstrAttrib struct {
	Tag    int
	Arity  int
	Update string
	Pack   string
}

func setConstrAttrib(temp string, cons ConstrAttrib) string {
	tmpl, err := template.New("test").Parse(string(temp))
	if err != nil {
		panic(err)
	}
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, cons)
	return doc.String()
}

type PackStruct struct {
	Ninstr int
	Tag    int
	Arity  int
}

func packAttrib(temp string, pck PackStruct) string {
	tmpl, err := template.New("test").Parse(string(temp))
	if err != nil {
		panic(err)
	}
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, pck)
	return doc.String()
}

type Relational struct {
	Ninstr   int
	Instr    string
	TrueTag  int
	FalseTag int
}

func setRelationalAttrib(temp string, rel Relational) string {
	tmpl, err := template.New("test").Parse(string(temp))
	if err != nil {
		panic(err)
	}
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, rel)
	return doc.String()
}

type Arithmatic struct { //Done
	Ninstr int
	Instr  string
}

func setArithAttrib(temp string, ari Arithmatic) string { //Done
	tmpl, err := template.New("test").Parse(string(temp))
	if err != nil {
		panic(err)
	}
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, ari)
	return doc.String()
}

type Inventory struct {
	N      core.Instruction
	Ninstr int
}

func setManyAttrib(b string, inv Inventory) string { //Done
	tmpl, err := template.New("test").Parse(string(b))
	if err != nil {
		panic(err)
	}
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, inv)
	return doc.String()
}

type GlobalAttr struct {
	Arity  Arity
	Name   core.Instruction
	Ninstr int
}

func setPushGlobalAttrib(b string, inv GlobalAttr) string { //Done
	tmpl, err := template.New("test").Parse(string(b))
	if err != nil {
		panic(err)
	}
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, inv)
	return doc.String()
}

type ScsAttrib struct { //Done
	Body string
	Name string
}

func setScAttrib(b string, inv ScsAttrib) string { //Done
	tmpl, err := template.New("test").Parse(string(b))
	if err != nil {
		panic(err)
	}
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, inv)
	return doc.String()
}

func foldl(arr []int) int { //Done
	tmp := 0
	for a := range arr {
		tmp = tmp + arr[a]
		fmt.Println(tmp, a)
	}
	return tmp
}
