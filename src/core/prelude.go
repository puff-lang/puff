package core

var x = [...]Name{"x"}

var PreludeDefs = Program{
	{"I", []Name{"x"}, EVar("x")},
	{"K", []Name{"x","y"}, EVar("x")},
	{"K1",[]Name{"x","y"}, EVar("y")},
	{"S", []Name{"f","g","x"}, EAp{EAp{EVar("f"), EVar("x")}, EAp{EVar("g"), EVar("x")}}},
	{"compose", []Name{"f","g","x"}, EAp{EVar("f"), EAp{EVar("g"), EVar("x")}}},
	{"twice", []Name{"f"}, EAp{ EAp{EVar("compose"), EVar("f")}, EVar("f")}},
}