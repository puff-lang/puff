package core

var x = [...]Name{"x"}

//STG Haskell Spineless Tagless GMachine

var PreludeDefs = Program{
	{"I", []Name{"x"}, EVar("x")},
	{"K", []Name{"x","y"}, EVar("x")},
	{"K1",[]Name{"x","y"}, EVar("y")},
	{"S", []Name{"f","g","x"}, EAp{EAp{EVar("f"), EVar("x")}, EAp{EVar("g"), EVar("x")}}},
	{"compose", []Name{"f","g","x"}, EAp{EVar("f"), EAp{EVar("g"), EVar("x")}}},
	{"twice", []Name{"f"}, EAp{ EAp{EVar("compose"), EVar("f")}, EVar("f")}},
}

type compiledPrimitives []GmCompiledSC

var compPrim = compiledPrimitives{
	GmCompiledSC{"add", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Add{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{"sub", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Sub{}, Update(2), Pop(2), Unwind{}}},
	// GmCompiledSC{"*", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Mul{}, Update(2), Pop(2), Unwind{}}},
	// GmCompiledSC{"/", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Div{}, Update(2), Pop(2), Unwind{}}},
	// GmCompiledSC{"negate", 2, GmCode{Push(0), Eval{}, Neg{}, Update(1), Pop(1), Unwind{}}},
	// GmCompiledSC{"==", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Eq{}, Update(2), Pop(2), Unwind{}}},
	// GmCompiledSC{"~=", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Ne{}, Update(2), Pop(2), Unwind{}}},
	// GmCompiledSC{"<", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Lt{}, Update(2), Pop(2), Unwind{}}},
	// GmCompiledSC{"<=", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Le{}, Update(2), Pop(2), Unwind{}}},
	// GmCompiledSC{">", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Gt{}, Update(2), Pop(2), Unwind{}}},
	// GmCompiledSC{">=", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Ge{}, Update(2), Pop(2), Unwind{}}},
	// GmCompiledSC{"if", 3, GmCode{Push(0), Eval{}, Cond{GmCode{Push(1)}, GmCode{Push(2)}}, Update(3), Pop(3), Unwind{}}},
}