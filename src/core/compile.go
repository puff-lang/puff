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
	fmt.Println("Globals", globals)
	fmt.Println("Heap at 0", heap.hNode[0])
	return GmState{GmOutput{}, initialCode(), InitStack(), initialDump(), GmVStack{}, heap, globals, stats}
}

func buildInitialHeap(p Program) (GmHeap, GmGlobals) {
	var compiled []GmCompiledSC
	gmHeap := HInitial()
	p = append(p, primitiveScs()...)
	for _, sc := range p {
		compiled = append(compiled, compileSc(sc))
	}
	// compiled = append(compPrim, compiled...)
	return mapAccuml(allocateSc, gmHeap, compiled)
}

//-------------------------------------------------------------------
//We must define the type for passing function as parameter
type allocates func(GmHeap, GmCompiledSC) (GmHeap, Object) 

// allocateSc implements allocates, returning GmHeap & Object
func allocateSc(gmh GmHeap, gCSC GmCompiledSC) (GmHeap, Object) {
	fmt.Println("No. of Args for", gCSC.Name, gCSC.Length)
	addr := gmh.HAlloc(NGlobal{gCSC.Length, gCSC.body})
	fmt.Println("SC: ", gCSC.Name, "stored at: ", addr)
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
	return GmCode{Pushglobal("main"), Eval{}}
}

func (sc GmCompiledSC) Body() GmCode {
	return sc.body
}


//Each SuperCombinator is compiled using compileSc which implements SC scheme
func compileSc(sc ScDefn) GmCompiledSC {
	fmt.Println("Compiling SC", sc.Name)
	var gmE = GmEnvironment{}

	for i,eString := range sc.Args {
		gmE = append(gmE, Environment{eString, i})
	}
	l := len(sc.Args)
	compileR := createCompileR(l)
	return GmCompiledSC{sc.Name, l, compileR(sc.Expr, gmE)}
}

//Each SuperCombinator is compiled using compileSc which implements SC scheme
func CompileSc(sc ScDefn) GmCompiledSC {
	var gmE = GmEnvironment{}

	for i, eString := range sc.Args {
		gmE = append(gmE, Environment{eString, i})
	}
	l := len(sc.Args)
	compileR := createCompileR(l)
	return GmCompiledSC{sc.Name, l, compileR(sc.Expr, gmE)}
}

func elem(name Name, assoc GmEnvironment) int {
	for _,obj := range assoc {
		if obj.Name == name  {
			return obj.Int
		}
	}
	return -1 //Default Value: null string
}

func createCompileR(d int) GmCompiler {
	return func(cexp CoreExpr, env GmEnvironment) GmCode {
		switch cexp.(type) {
		case ELet:
			fmt.Println("Compiling Let in compileR")
			expr := cexp.(ELet)
			if expr.IsRec {
				return compileLetrec(GmCode{}, createCompileR(d + len(expr.Defns)), expr.Defns, expr.Body, env)
			} else {
				return compileLet(GmCode{}, createCompileR(d + len(expr.Defns)), expr.Defns, expr.Body, env)
			}

		case ECaseSimple:
			expr := cexp.(ECaseSimple)
			i := compileD(createCompileR(d + 1), expr.Alts, argOffset(1, env))
			return append(compileE(expr.Body, env), CasejumpSimple(i))

		case ECaseConstr:
			expr := cexp.(ECaseConstr)
			i := compileD(createCompileR(d + 1), expr.Alts, argOffset(1, env))
			return append(compileE(expr.Body, env), CasejumpConstr(i)	)

		default:
			fmt.Println("Default in compileR with d, ", d)
			fmt.Println("Env ", env)
			return append(compileE(cexp, env), []Instruction{Update(d), Pop(d), Unwind{}}...)
		}
	} 
}

// func compilerRX(d int, cexp CoreExpr, env GmEnvironment) GmCode {
// 	switch cexp.(type) {
// 		case ELet:
// 			expr := cexp.(ELet)
// 			if expr.IsRec {
// 				return compileLetrec(GmCode{}, createCompileR(d + len(expr.Defns)), expr.Defns, expr.Body, env)
// 			} else {
// 				return compileLet(GmCode{}, createCompileR(d + len(expr.Defns)), expr.Defns, expr.Body, env)
// 			}
// 		// case ECaseSimple:
// 		// 	expr := cexp.(ECaseSimple)

// 		// case ECaseConstr:
// 		// 	expr := cexp.(ECaseConstr)

// 		default:
// 			inst := []Instruction{}
// 			cC := compileE(cexp,env)
// 			for _,obj := range cC {
// 				inst = append(inst, obj)
// 			}
// 			inst = append(inst, Update(d))
// 			inst = append(inst, Pop(d))
// 			inst = append(inst, Unwind{})
// 			return inst	
// 	}
// }

type GmCompiler func(CoreExpr, GmEnvironment) (GmCode)


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

		case ECaseSimple:
			expr := cexp.(ECaseSimple)
			i := compileD(compileE, expr.Alts, argOffset(1, env))
			return append(compileE(expr.Body, env), CasejumpSimple(i))

		case ECaseConstr:
			expr := cexp.(ECaseConstr)
			i := compileD(compileE, expr.Alts, argOffset(1, env))
			fmt.Println("CompileD Done")
			return append(compileE(expr.Body, env), CasejumpConstr(i))
		
		case EAp:
			expr := cexp.(EAp)
			fmt.Println("EAp 1")
			switch expr.Left.(type) {
				case EAp:
					fmt.Println("EAp 2")
					expr1 := expr.Left.(EAp)
					switch expr1.Left.(type) {
						case EVar:
							fmt.Println("Compiling EVar 1")
							expr2 := expr1.Left.(EVar)
							fmt.Println(aHasKey(built, string(expr2)), expr2, " Buildyadic:  ",built)
							if aHasKey(built, string(expr2)) {
								fmt.Println("Going for CompileB")
								return append(compileB(expr, env), intOrBool(Name(expr2)))
							} else {
								return append(compileC(expr, env), Eval{})
							}
						case EAp:
							ifApExpr := expr1.Left.(EAp)
							switch ifApExpr.Left.(type) {
								case EVar:
									name := ifApExpr.Left.(EVar)
									if name == "if" {
										fmt.Println("Inside if else")
										result := GmCode{}
										result = append(result, compileE(ifApExpr.Body, env)...)
										fmt.Println("Compiling then and else body")
										instn := CasejumpConstr{CasejumpObj{trueTag, compileE(expr1.Body, env)}, CasejumpObj{falseTag, compileE(expr.Body, env)}}
										result = append(result, GmCode{instn}...)
										return result
									} else {
										fmt.Println(" 227 Special Case CompileE ")
										return append(compileC(expr,env), Eval{})
									}
								default:
									fmt.Println(" 231 Default CompileE ")
									return append(compileC(expr,env), Eval{})
							}
							
						default:
							fmt.Println("210 CompileE expression syntax")
							return append(compileC(expr, env), Eval{})
					}

				case EVar:
					expr1 := expr.Left.(EVar)
					if expr1 == "negate" {
						return append(compileB(expr.Body, env), GmCode{MkInt{}}...)
					} else {
						fmt.Println("EAp -> EVar ", expr1)
						// return GmCode{} //Don't no what is right condition
						return append(compileC(expr, env), Eval{})		
					}

				default:
					fmt.Println("223 CompileE expression syntax")
					return append(compileC(expr, env), Eval{})
			}

		default:
			expr := cexp
			fmt.Println("229 CompileE expression syntax", expr)
			return append(compileC(expr, env), Eval{})		
	}
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
										fmt.Println(" Inside if else")
										result := compileE(ifApExpr.Body, env)
										instn := CasejumpConstr{CasejumpObj{trueTag, compileB(expr1.Body, env)}, CasejumpObj{falseTag, compileB(expr.Body, env)}}
										return append(result, GmCode{instn}...)
									} else {
										fmt.Println(" 299 Special Case CompileB")
										return append(compileC(expr,env), Eval{})
									}
								default:
									fmt.Println(" 303 Default CompileB")
									return append(compileC(expr,env), Eval{})
							}

						default:
							return append(compileE(expr, env), Get{})
					}

				case EVar:
					expr1 := expr.Left.(EVar)
					if expr1 == "negate" {
						return append(compileB(expr.Body, env), GmCode{Neg{}}...)
					} else {
						return append(compileE(expr, env), Get{}) //Don't no what is right condition
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
}

//generates code which creates the graph of e in env ρ,leaving a pointer to it on top of the stack
func compileC(cexp CoreExpr, env GmEnvironment) GmCode {
	switch cexp.(type) {
        case EVar:
        	expr := cexp.(EVar)
        	n := elem(Name(expr), env)
        	fmt.Println("Env during EVar ", expr, " in compileC, ", env, "n, ", n)
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
			fmt.Println("In EAp for compileC")
			expr := cexp.(EAp)
			var gmC = GmCode{}
			fmt.Println("Compiling Body: ", expr.Body)
			gmC = append(gmC, compileC(expr.Body, env)...)
			fmt.Println("Compiling left of EAp")
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

		case ECaseSimple:
			expr := cexp.(ECaseSimple)
			i := compileD(compileE, expr.Alts, argOffset(1, env))
			return append(compileE(expr.Body, env), CasejumpSimple(i))

		case ECaseConstr:
			expr := cexp.(ECaseConstr)
			i := compileD(compileE, expr.Alts, argOffset(1, env))
			return append(compileE(expr.Body, env), CasejumpConstr(i))
		
		default:
			fmt.Println("Compilation scheme for the following expression does not exist.")
			return GmCode{}
    }
}

func compileD(comp GmCompiler, alts []CoreAlt, env GmEnvironment) []CasejumpObj {
	fmt.Println("In compileD")
	var list []CasejumpObj
	for _, alt := range alts {
		fmt.Println("Compiling Alt", alt)
		list = append(list, compileA(comp, alt, env))
	} 

	return list
} 

func compileA(comp GmCompiler, alt CoreAlt, env GmEnvironment) CasejumpObj {
	return CasejumpObj{alt.Num, comp(alt.Expr, env)}
}


func compileLet(finalInst GmCode, comp GmCompiler, defs []Defn, expr CoreExpr, env GmEnvironment) GmCode {
	envdash := compileArgs(defs, env) // Creating New Environment
	fmt.Println("New Env: ", envdash)
	gmC := GmCode{}
	gmC = append(gmC, compileDefs(defs, env)...)
	gmC = append(gmC,  comp(expr, envdash)...)
	return append(gmC, finalInst...)
}

func compileDefs(defns []Defn, env GmEnvironment) GmCode {
	if len(defns) <= 0 {
		return GmCode{}
	}
	
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
	fmt.Println("compileArgs: env:", env)
	n := len(defns)
	var gmE GmEnvironment
	for _, defn := range defns {
		tmpEnv := Environment{defn.Var, n - 1}
		gmE = append(gmE, tmpEnv)
		n = n - 1
	}
	return append(gmE, argOffset(len(defns), env)...)
}

func argOffset(n int, env GmEnvironment) GmEnvironment {
	fmt.Println("Offsetting args for,", env, " with n as ", n)
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

func intOrBool(name Name) Instruction {
	/*
	if nm == "+" || nm == "-" || nm == "*" || nm == "/" || nm == "%" {
		return MkInt{}
	}  else if nm == "==" || nm ==">=" || nm == ">" || nm =="<" || nm =="<=" || nm == "!=" {
		return MkBool{}
	} else {
		tp := "Name: " + nm + " is not a built-in operator"
		return Error(tp)
	}
	*/

	for _, bd := range builtinDyadicInt {
		if bd.Name == string(name) {
			return MkInt{}
		}
	}

	for _, bd := range builtinDyadicBool {
		if bd.Name == string(name) {
			return MkBool{}
		}
	}

	panic("Name: " + name + " is not a built-in operator")
}
