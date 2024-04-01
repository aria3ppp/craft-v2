package craft

import (
	_ "embed"

	craft_error "github.com/aria3ppp/craft/error"
)

//go:embed program.template
var programTemplate string

type TemplateDate struct {
	OutputFilePath  string
	ValueDefinition string
	SourceName      string
	TypeName        string
	SourcePosition  craft_error.Position
	Macro           TemplateDataMacro
	Package         TemplateDataPackage
}

type TemplateDataMacro struct {
	Name       string
	ImportPath string
	Input      string
	Position   craft_error.Position
}

type TemplateDataPackage struct {
	RelativePath         string
	GoFile               string
	ImportPathDefinition string
	Name                 string
}
