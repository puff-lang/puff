package core

import (
	"fmt"
	"reflect"
)


type Environment struct{
	Name Name
	Int int
}

type GmEnvironment []Environment

type GmCompiledSC struct{
	Name   Name
	Length int
	body   GmCode
}


func Compile(p Program) GmState {
	var stats GmStats = 0
	fmt.Println(preCompiledScs())
	heap, globals := buildInitialHeap(p)
	return GmState{GmOutput{}, initialCode(), InitStack(), initialDump(), GmVStack{}, heap, globals, stats}
}


func buildInitialHeap(p Program) (GmHeap, GmGlobals) {
	var compiled []GmCompiledSC
	gmHeap := HInitial()
	// p = append(p, primitiveScs()...)
	for _, sc := range p {
		compiled = append(compiled, compileSc(sc))
	}
	compiled = append(compPrim, compiled...)
	return mapAccuml(allocateSc, gmHeap, compiled)
}

//-------------------------------------------------------------------
//We must define the type for passing function as parameter
type allocates func(GmHeap, GmCompiledSC) (GmHeap, Object) 

// allocateSc implements allocates, returning GmHeap & Object
func allocateSc(gmh GmHeap, gCSC GmCompiledSC) (GmHeap, Object) {
	addr := gmh.HAlloc(NGlobal{gCSC.Length, gCSC.body})
	// fmt.Println("Allocated Heap: ",gmh, "GM Addresses: ",addr)
	return gmh, Object{gCSC.Name, addr}
}

func mapAccuml(f allocates, acc GmHeap, list []GmCompiledSC) (GmHeap, GmGlobals) {
	acc1 := acc
	xsdash := GmGlobals{}
	var xdash Object

	for _, sc := range list {
		acc1, xdash = f(acc1, sc)
		xsdash = append(GmGlobals{xdash}, xsdash...)
	}
	return acc1, xsdash
}

func initialCode() GmCode {
	fmt.Println("Got Eval")
	return GmCode{Pushglobal("main"), Eval{}}
}

func (sc GmCompiledSC) Body() GmCode {
	return sc.body
}


//Each SuperCombinator is compiled using compileSc which implements SC scheme
func compileSc(sc ScDefn) GmCompiledSC {
	var gmE = GmEnvironment{}

	for i,eString := range sc.Args {
		gmE = append(gmE, Environment{eString, i})
	}
	l := len(sc.Args)
	fmt.Println("hello",l)
	return GmCompiledSC{sc.Name, l, compilerRX(l, sc.Expr, gmE)}
}

//Each SuperCombinator is compiled using compileSc which implements SC scheme
func CompileSc(sc ScDefn) GmCompiledSC {
	var gmE = GmEnvironment{}

	for i,eString := range sc.Args {
		gmE = append(gmE, Environment{eString, i})
	}
	l := len(sc.Args)
	fmt.Println("hello: ", l)
	return GmCompiledSC{sc.Name, l, compilerRX(l, sc.Expr, gmE)}
}

func elem(name Name, assoc GmEnvironment) int {
	for _,obj := range assoc {
		if obj.Name == name  {
			return obj.Int
		}
	}
	return -1 //Default Value: null string
}

func compilerRX(d int, cexp CoreExpr, env GmEnvironment) GmCode {
	switch cexp.(type) {
		case ELet:
			expr := cexp.(ELet)
			if expr.IsRec {
				return compileLetrec(GmCode{}, compilerR, expr.Defns, expr.Body, env)
			} else {
				return compileLet(GmCode{}, compilerR, expr.Defns, expr.Body, env)
			}
		// case ECaseSimple:
		// 	expr := cexp.(ECaseSimple)

		// case ECaseConstr:
		// 	expr := cexp.(ECaseConstr)

		default:
			inst := []Instruction{}
			cC := compileE(cexp,env)
			for _,obj := range cC {
				inst = append(inst, obj)
			}
			inst = append(inst, Update(d))
			inst = append(inst, Pop(d))
			inst = append(inst, Unwind{})
			return inst	
	}
}

type GmCompiler func(CoreExpr, GmEnvironment) (GmCode)

func compilerR(cexp CoreExpr, env GmEnvironment) GmCode {
	switch cexp.(type) {
		case ELet:
			expr := cexp.(ELet)
			if expr.IsRec {
				return compileLetrec(GmCode{}, compilerR, expr.Defns, expr.Body, env)
			} else {
				return compileLet(GmCode{}, compilerR, expr.Defns, expr.Body, env)
			}

		default:
			return compilerRX(0, cexp, env)	
	}
}

func compileE(cexp CoreExpr, env GmEnvironment) GmCode { // 2 Conditions :TODO
	switch cexp.(type) {
		case ENum:
			fmt.Println("ENum of compileEEE")
			expr := cexp.(ENum)
			if expr.IsInt {
				return GmCode{Pushint(expr.Int64)}
			} else  { //if expr.IsUint
				return GmCode{Pushint(expr.Uint64)}
			}
		
		case EChar:
			fmt.Println("EChar of compileEEE")
			expr := cexp.(EChar)
			return GmCode{Pushchar(expr)}

		case ELet:
			expr := cexp.(ELet)
			if expr.IsRec {
				return compileLetrec(GmCode{Slide(len(expr.Defns))}, compileE, expr.Defns, expr.Body, env)
			} else {
				return compileLet(GmCode{Slide(len(expr.Defns))}, compileE, expr.Defns, expr.Body, env)
			}
		
		case EAp:
			expr := cexp.(EAp)
			fmt.Println("EAp 1")
			switch expr.Left.(type) {
				case EAp:
					fmt.Println("EAp 2")
					expr1 := expr.Left.(EAp)
					switch expr1.Left.(type) {
						case EVar:
							fmt.Println("EVar 1")
							expr2 := expr1.Left.(EVar)
							fmt.Println(aHasKey(built, string(expr2)), " Buildyadic:  ",built)
							if aHasKey(built, string(expr2)) {
								fmt.Println("Going for CompileB")
								return append(compileB(expr, env), intOrBool(Name(expr2)))
							} else {
								return append(compileC(expr,env), Eval{})
							}
						case EAp:
							ifApExpr := expr1.Left.(EAp)
							switch ifApExpr.Left.(type) {
								case EVar:
									name := ifApExpr.Left.(EVar)
									if name == "if" {
										result := GmCode{}
										result = append(result, compileE(ifApExpr.Body, env)...)
										instn := CasejumpConstr{{trueTag, compileE(expr1.Body, env)}, {falseTag, compileE(expr.Body, env)}}
										result = append(result, GmCode{instn}...)
									}
							}
							
						default:
							fmt.Println("CompileE expression syntax")
							return append(compileC(expr, env), Eval{})
					}

				case EVar:
					expr1 := expr.Left.(EVar)
					if expr1 == "negate" {
						return append(compileB(expr.Body, env), GmCode{MkInt{}}...)
					} else {
						return GmCode{} //Don't no what is right condition
					}

				default:
					fmt.Println("CompileE expression syntax")
					return append(compileC(expr, env), Eval{})
			}

		default:
			expr := cexp
			fmt.Println("CompileE expression syntax")
			return append(compileC(expr, env), Eval{})		
	}
	return GmCode{}
}


func compileB(cexp CoreExpr, env GmEnvironment) GmCode { //All Cases Covered for compileB
	switch cexp.(type) {
		case ENum:
			expr := cexp.(ENum)
			if expr.IsInt {
				return GmCode{Pushbasic(expr.Int64)}
			} else  { //if expr.IsUint
				return GmCode{Pushbasic(expr.Uint64)}
			}
		
		case EAp:
			expr := cexp.(EAp)
			switch expr.Left.(type) {
				case EAp:
					expr1 := expr.Left.(EAp)
					switch expr1.Left.(type) {
						case EVar:
							expr2 := expr1.Left.(EVar)
							if aHasKey(built, string(expr2)) {
								result := GmCode{}
								result = append(result, compileB(expr.Body, env)...)
								result = append(result, compileB(expr1.Body, env)...)
								result = append(result, aLookup(built, string(expr2)))
								return result
							} else {
								return append(compileE(expr,env), Get{})
							}

						case EAp:
							ifApExpr := expr1.Left.(EAp)
							switch ifApExpr.Left.(type) {
								case EVar:
									name := ifApExpr.Left.(EVar)
									if name == "if" {
										result := GmCode{}
										result = append(result, compileB(ifApExpr.Body, env)...)
										instn := CasejumpConstr{{trueTag, compileB(expr1.Body, env)}, {falseTag, compileE(expr.Body, env)}}
										result = append(result, GmCode{instn}...)
									}
							}

						default:
							return append(compileE(expr, env), Get{})
					}

				case EVar:
					expr1 := expr.Left.(EVar)
					if expr1 == "negate" {
						return append(compileB(expr.Body, env), GmCode{Neg{}}...)
					} else {
						return GmCode{} //Don't no what is right condition
					}

				default:
					return append(compileE(expr, env), Get{})
			}

		case ELet:
			expr := cexp.(ELet)
			if expr.IsRec {
				return compileLetrec(GmCode{Pop(len(expr.Defns))}, compileB, expr.Defns, expr.Body, env)
			} else {
				return compileLet(GmCode{Pop(len(expr.Defns))}, compileB, expr.Defns, expr.Body, env)
			}

		default:
			expr := cexp
			return append(compileE(expr, env), Get{})
	}
	return GmCode{}
}

//generates code which creates the graph of e in env ρ,leaving a pointer to it on top of the stack
func compileC(cexp CoreExpr, env GmEnvironment) GmCode {
	switch cexp.(type) {
        case EVar:
        	expr := cexp.(EVar)
        	n := elem(Name(expr), env)
        	if n != -1 {
        		return GmCode{Push(n)}
        	} else {
        		return GmCode{Pushglobal(Name(expr))}
        	}

		case ENum:
			expr := cexp.(ENum)
			if expr.IsInt {
				return GmCode{Pushint(expr.Int64)}
			} else if expr.IsUint {
				return GmCode{Pushint(expr.Uint64)}
			}
			return GmCode{Pushint(42)} // TODO

		case EChar:
			expr := cexp.(EChar)
			return GmCode{Pushchar(expr)}

		case EConstr:
			expr := cexp.(EConstr)
			return GmCode{Pushconstr{expr.Tag, expr.Arity}}

		case EAp:
			expr := cexp.(EAp)
			var gmC = GmCode{}
			gmC = append(gmC, compileC(expr.Body, env)...)
			gmC = append(gmC, compileC(expr.Left, argOffset(1, env))...)
			gmC = append(gmC, Mkap{})
			return gmC

		case ELet:
			expr := cexp.(ELet)
			if expr.IsRec {
				return compileLetrec(GmCode{Slide(len(expr.Defns))}, compileC, expr.Defns, expr.Body, env)
			} else {
				return compileLet(GmCode{Slide(len(expr.Defns))}, compileC, expr.Defns, expr.Body, env)
			}

		default:
			fmt.Println("Compilation scheme for the following expression does not exist.")
			return GmCode{}
    }
}


func compileLet(finalInst GmCode, comp GmCompiler, defs []Defn, expr CoreExpr, env GmEnvironment) GmCode {
	envdash := compileArgs(defs, env) // Creating New Environment
	gmC := GmCode{}
	gmC = append(gmC, compileDefs(defs, envdash)...)
	gmC = append(gmC,  comp(expr, envdash)...)
	return append(gmC, finalInst...)
}

func compileDefs(defns []Defn, env GmEnvironment) GmCode {
	envdash := env
	gmC := GmCode{}
	for _, defn := range defns {
		gmC = append(gmC, compileC(defn.Expr, envdash)...)
		envdash = argOffset(1, envdash)
	}
	return gmC
}

func compileLetrec(finalInst GmCode, comp GmCompiler, defs []Defn, expr CoreExpr, env GmEnvironment) GmCode {
	envdash := compileArgs(defs, env) // Creating New Environment
	n := len(defs)
	gmC := GmCode{Alloc(n)}
	gmC = append(gmC, compileRecDefs(n, defs, envdash)...)
	gmC = append(gmC, Update(0))
	gmC = append(gmC,  comp(expr, envdash)...)
	return append(gmC, finalInst...)
}

func compileRecDefs(n int, defns []Defn, env GmEnvironment) GmCode {
	envdash := env
	gmC := GmCode{}
	for _, defn := range defns {
		gmC = append(gmC, compileC(defn.Expr, envdash)...)
		gmC = append(gmC, Update(n-1))
		envdash = argOffset(1, envdash)
		n = n - 1
	}
	return gmC
}


func compileArgs(defns []Defn, env GmEnvironment) (GmEnvironment) {
	n := len(defns)
	var gmE GmEnvironment
	for _, defn := range defns {
		tmpEnv := Environment{defn.Var, n-1}
		gmE = append(gmE, tmpEnv)
		n = n - 1
	}
	return append(gmE, argOffset(len(defns), env)...)
}

func argOffset(n int, env GmEnvironment) GmEnvironment {	
	var gmE GmEnvironment
	for _,obj := range env {
		tmpEnv := Environment{obj.Name, obj.Int + n}
		gmE = append(gmE, tmpEnv)
	}
	return gmE
}

func PrintBody(body GmCode) {
	for _, inst := range body {
		fmt.Print(reflect.TypeOf(inst), inst, "  ")
	}
	fmt.Println()
}

func intOrBool(nm Name) Instruction{
	if nm == "+" || nm == "-" || nm == "*" || nm == "/" || nm == "%" {
		return MkInt{}
	}  else if nm == "==" || nm ==">=" || nm == ">" || nm =="<" || nm =="<=" || nm == "!=" {
		return MkBool{}
	} else {
		tp := "Name: " + nm + " is not a built-in operator"
		return Error(tp)
	}
}
















