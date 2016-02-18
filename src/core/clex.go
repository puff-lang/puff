package core


type Token string

type clexer struct{
	input string
	tokens chan Token

	start      int
	pos        int
	width      int
}

func clex (text string) *clexer{
	cl := &lexer{
		input:  input,
		tokens: make(chan Token),
	}
	go cl.run()
	return cl
}

func (clex *cl) run(){

}

func () isWhiteSpace() {
	if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
		return false
	} else {
		return 
	}
}

func syntax (*tok Token[]) CoreProgram{

} 

func parse (filename string) CoreProgram{
	return syntax(lex(filename))
}