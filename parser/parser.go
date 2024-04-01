package parser

import (
	"github.com/alecthomas/participle/v2"
)

type MacroAST struct {
	Package string `parser:" '#' @Ident"`
	Macro   string `parser:" '.' @Ident"`
	Input   string `parser:"( '(' @RawString ')' )?"`
}

func NewMacroASTParser() (*participle.Parser[MacroAST], error) {
	return participle.Build[MacroAST](
		participle.Unquote("RawString"),
	)
}
