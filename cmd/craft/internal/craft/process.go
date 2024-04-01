package craft

import (
	"go/token"

	craft_parser "github.com/aria3ppp/craft/parser"
)

type Process struct {
	// GenDecl      *ast.GenDecl
	// Spec         *ast.TypeSpec
	// TypePosition token.Position
	SourceName         string
	UnexportedTypeName string
	SourcePosition     token.Position
	Macros             []*Macro
}

type Macro struct {
	AST           *craft_parser.MacroAST
	MacroPosition token.Position
}
