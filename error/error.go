package error

import (
	"fmt"
	"path/filepath"
)

type Error struct {
	Msg            string
	RelativePath   string
	GoFile         string
	MacroPosition  Position
	SourcePosition Position
	Kind           Kind
}

var _ error = (*Error)(nil)

func (e Error) Error() (errorString string) {
	gofileRelativePath := filepath.Join(e.RelativePath, e.GoFile)

	switch e.Kind {
	default:
		errorString = fmt.Sprintf("%s:%d:%d: %s %s:%d:%d", gofileRelativePath, e.SourcePosition.Line, e.SourcePosition.Column, e.Msg, gofileRelativePath, e.MacroPosition.Line, e.MacroPosition.Column)
	case KindProgram:
		errorString = fmt.Sprintf("%s:%d:%d: %s", gofileRelativePath, e.MacroPosition.Line, e.MacroPosition.Column, e.Msg)
	}

	return
}
