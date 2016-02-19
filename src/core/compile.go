package core

func Compile(p Program) GmState {
	heap, globals := buildInitialHeap(p)
	return GmState{initialCode, [], heap, globals, statInitial}
}

func buildInitialHeap(p Program) (GmHeap, GmGlobals) {
	var compiled []GmCompiledSC

	for _, sc := range p {
		compiled = append(compiled, sc)
	}

	// mapAccuml allocateSc hInitial compiled
	for _, compiledSc := range compiled {
		allocateSc(compiledSc)	
	}
}

func allocateSc() {}

const initialCode = 