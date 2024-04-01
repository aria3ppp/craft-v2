package craft

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/aria3ppp/craft/cmd/craft/internal/comment"
	craft_error "github.com/aria3ppp/craft/error"
	craft_parser "github.com/aria3ppp/craft/parser"

	"github.com/alecthomas/participle/v2"
)

type Craft struct {
	Context        *Context
	CurrentASTFile *ast.File
	FileSet        *token.FileSet
	Parser         *participle.Parser[craft_parser.MacroAST]

	// nil is a valid default value for the following fields

	Processes []Process
	Errs      []craft_error.Error

	processesMu sync.Mutex
	errsMu      sync.Mutex
}

func (c *Craft) HandleMacrosOnSpec(
	genDecl *ast.GenDecl,
	spec ast.Spec,
) {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		c.HandleMacrosOnTypeSpec(genDecl, s)
	case *ast.ValueSpec:
		c.HandleMacrosOnValueSpec(genDecl, s)
	}
}

func (c *Craft) HandleMacrosOnTypeSpec(
	genDecl *ast.GenDecl,
	spec *ast.TypeSpec,
) {
	switch spec.Type.(type) {
	case *ast.InterfaceType:
		return
	}

	typeName := spec.Name.Name

	// skip if type name is not exported
	if !token.IsExported(typeName) {
		return
	}

	var (
		doc     = genDecl.Doc
		specPos = spec.Pos()
	)

	if genDecl.Lparen.IsValid() {
		doc = spec.Doc
	}

	if doc == nil || len(doc.List) == 0 {
		return
	}

	c.HandleMacroOnSource(
		typeName,
		"",
		specPos,
		doc.List,
	)
}

func (c *Craft) HandleMacrosOnValueSpec(
	genDecl *ast.GenDecl,
	spec *ast.ValueSpec,
) {
	var typeName string

	switch typ := spec.Type.(type) {
	default:
		return
	case *ast.Ident:
		typeName = typ.Name
	}

	// skip if type is exported
	if token.IsExported(typeName) {
		return
	}

	varName := spec.Names[0].Name

	// skip if var name is not exported
	if !token.IsExported(varName) {
		return
	}

	var (
		doc     = genDecl.Doc
		specPos = spec.Pos()
	)

	if genDecl.Lparen.IsValid() {
		doc = spec.Doc
	}

	if doc == nil || len(doc.List) == 0 {
		return
	}

	c.HandleMacroOnSource(
		varName,
		typeName,
		specPos,
		doc.List,
	)
}

// TODO: is 'HandleMacro' a good name?
func (c *Craft) HandleMacroOnSource(
	sourceName string,
	unexportedTypeName string,
	sourcePos token.Pos,
	comments []*ast.Comment,
) {
	iter := comment.Iter{
		Comments: comments,
	}

	sourcePosition := c.FileSet.Position(sourcePos)

	process := Process{
		// GenDecl:      genDecl,
		// Spec:         spec,
		// TypePosition: typePosition,
		SourceName:         sourceName,
		UnexportedTypeName: unexportedTypeName,
		SourcePosition:     sourcePosition,
		Macros:             nil,
	}

	for comment := iter.Next(); comment != nil; comment = iter.Next() {
		commentPosition := c.FileSet.Position(comment.Pos() + token.Pos(comment.StartOffset))

		macroAST, err := c.Parser.ParseString("", comment.Text)
		if err != nil {
			if macroAST != nil && macroAST.Macro != "" {
				var participleError participle.Error
				errors.As(err, &participleError) // SAFETY: participle errors are all participle.Error

				participleErrorPosition := participleError.Position()
				macroErrorPosition := c.FileSet.Position(token.Pos(commentPosition.Offset) + 1 + token.Pos(participleErrorPosition.Offset))

				c.addError(
					craft_error.Error{
						Msg:            participleError.Message(),
						RelativePath:   c.Context.RelativePath,
						GoFile:         c.Context.GoFile,
						MacroPosition:  craft_error.PositionFromToken(macroErrorPosition),
						SourcePosition: craft_error.PositionFromToken(sourcePosition),
					},
				)
			}

			continue
		}

		if c.Context.PackageImport(macroAST.Package) == "" {
			pkgIndex := strings.Index(comment.Text, macroAST.Package)
			macroErrorPosition := c.FileSet.Position(token.Pos(commentPosition.Offset) + 1 + token.Pos(pkgIndex))

			c.addError(
				craft_error.Error{
					Msg:            fmt.Sprintf("package %q not defined", macroAST.Package),
					RelativePath:   c.Context.RelativePath,
					GoFile:         c.Context.GoFile,
					MacroPosition:  craft_error.PositionFromToken(macroErrorPosition),
					SourcePosition: craft_error.PositionFromToken(sourcePosition),
				},
			)

			continue
		}

		if !token.IsExported(macroAST.Macro) {
			macroIndex := strings.Index(comment.Text, "."+macroAST.Macro)
			macroErrorPosition := c.FileSet.Position(token.Pos(commentPosition.Offset) + 1 + token.Pos(macroIndex) + 1)

			c.addError(
				craft_error.Error{
					Msg:            fmt.Sprintf("unexported macro %q is not supported", macroAST.Macro),
					RelativePath:   c.Context.RelativePath,
					GoFile:         c.Context.GoFile,
					MacroPosition:  craft_error.PositionFromToken(macroErrorPosition),
					SourcePosition: craft_error.PositionFromToken(sourcePosition),
				},
			)

			continue
		}

		poundIndex := strings.Index(comment.Text, "#")
		macroPosition := c.FileSet.Position(token.Pos(commentPosition.Offset) + 1 + token.Pos(poundIndex))

		process.Macros = append(
			process.Macros,
			&Macro{
				AST:           macroAST,
				MacroPosition: macroPosition,
			},
		)
	}

	if len(process.Macros) > 0 {
		c.addProcess(process)
	}
}

func (c *Craft) HandleProcess(
	process Process,
) {
	// DEBUG: remove this
	{
		fp := filepath.Join(c.Context.RelativePath, c.Context.GoFile)
		fmt.Printf("%s:%d:%d:\n", fp, process.SourcePosition.Line, process.SourcePosition.Column)

		for _, m := range process.Macros {
			fmt.Printf("\t%s:%d:%d: %s.%s(`%s`)\n", fp, m.MacroPosition.Line, m.MacroPosition.Column, m.AST.Package, m.AST.Macro, m.AST.Input)
		}
	}

	for _, macro := range process.Macros {
		c.GenerateProgram(process, macro)
	}
}

func (c *Craft) GenerateProgram(
	process Process,
	macro *Macro,
) {
	typeName := process.SourceName

	if process.UnexportedTypeName != "" {
		typeName = process.UnexportedTypeName
	}

	dirname := strings.ToLower(fmt.Sprintf("%d_%s_%s_%s", time.Now().UnixNano(), typeName, macro.AST.Package, macro.AST.Macro))
	dirPath := filepath.Join(c.Context.PWD, dirname)

	if err := os.Mkdir(dirPath, 0o755); err != nil {
		fmt.Println(craft_error.Error{
			Msg:            fmt.Sprintf("[INTERNAL ERROR] [file a bug] failed to mkdir %s: %s", dirPath, err.Error()),
			RelativePath:   c.Context.RelativePath,
			GoFile:         c.Context.GoFile,
			MacroPosition:  craft_error.PositionFromToken(macro.MacroPosition),
			SourcePosition: craft_error.PositionFromToken(process.SourcePosition),
		}.Error())

		return
	}

	defer func() {
		if err := os.RemoveAll(dirPath); err != nil {
			fmt.Println(craft_error.Error{
				Msg:            fmt.Sprintf("[INTERNAL ERROR] [file a bug] failed to remove dir %s: %s", dirPath, err.Error()),
				RelativePath:   c.Context.RelativePath,
				GoFile:         c.Context.GoFile,
				MacroPosition:  craft_error.PositionFromToken(macro.MacroPosition),
				SourcePosition: craft_error.PositionFromToken(process.SourcePosition),
			}.Error())
		}
	}()

	tmplt, err := template.New("").Parse(programTemplate)
	if err != nil {
		fmt.Println(craft_error.Error{
			Msg:            fmt.Sprintf("[INTERNAL ERROR] [file a bug] failed to template.New: %s", err.Error()),
			RelativePath:   c.Context.RelativePath,
			GoFile:         c.Context.GoFile,
			MacroPosition:  craft_error.PositionFromToken(macro.MacroPosition),
			SourcePosition: craft_error.PositionFromToken(process.SourcePosition),
		}.Error())

		return
	}

	bytesBuffer := bytes.NewBuffer(make([]byte, 0, len(programTemplate)))
	outputFilePath := filepath.Join(
		c.Context.RelativePath,
		fmt.Sprintf("%s_%s_%s.crafted.go", typeName, macro.AST.Package, macro.AST.Macro),
	)

	var valueDefinition string

	if process.UnexportedTypeName != "" {
		valueDefinition = "= "
	}

	currentPkgImportAlias := "typepkg"
	valueDefinition += fmt.Sprintf("%s.%s", currentPkgImportAlias, process.SourceName)
	pkgImportPathDefinition := fmt.Sprintf("%s \"%s\"", currentPkgImportAlias, c.Context.CurrentPkgImportPath)

	data := TemplateDate{
		OutputFilePath:  outputFilePath,
		ValueDefinition: valueDefinition,
		SourceName:      process.SourceName,
		TypeName:        typeName,
		SourcePosition:  craft_error.PositionFromToken(process.SourcePosition),
		Macro: TemplateDataMacro{
			Name:       macro.AST.Macro,
			ImportPath: c.Context.PackageImport(macro.AST.Package),
			Input:      macro.AST.Input,
			Position:   craft_error.PositionFromToken(macro.MacroPosition),
		},
		Package: TemplateDataPackage{
			RelativePath:         c.Context.RelativePath,
			GoFile:               c.Context.GoFile,
			ImportPathDefinition: pkgImportPathDefinition,
			Name:                 c.CurrentASTFile.Name.Name,
		},
	}

	if err = tmplt.Execute(bytesBuffer, data); err != nil {
		fmt.Println(craft_error.Error{
			Msg:            fmt.Sprintf("[INTERNAL ERROR] [file a bug] failed to template.Execute: %s", err),
			RelativePath:   c.Context.RelativePath,
			GoFile:         c.Context.GoFile,
			MacroPosition:  craft_error.PositionFromToken(macro.MacroPosition),
			SourcePosition: craft_error.PositionFromToken(process.SourcePosition),
		}.Error())

		return
	}

	programPath := filepath.Join(dirPath, "program.go")

	programFile, err := os.Create(programPath)
	if err != nil {
		fmt.Println(craft_error.Error{
			Msg:            fmt.Sprintf("[INTERNAL ERROR] [file a bug] failed to create %s: %s", programPath, err.Error()),
			RelativePath:   c.Context.RelativePath,
			GoFile:         c.Context.GoFile,
			MacroPosition:  craft_error.PositionFromToken(macro.MacroPosition),
			SourcePosition: craft_error.PositionFromToken(process.SourcePosition),
		}.Error())

		return
	}

	defer func() {
		if err := programFile.Close(); err != nil {
			fmt.Println(craft_error.Error{
				Msg:            fmt.Sprintf("[INTERNAL ERROR] [file a bug] failed to close %s: %s", programPath, err.Error()),
				RelativePath:   c.Context.RelativePath,
				GoFile:         c.Context.GoFile,
				MacroPosition:  craft_error.PositionFromToken(macro.MacroPosition),
				SourcePosition: craft_error.PositionFromToken(process.SourcePosition),
			}.Error())
		}
	}()

	if _, err := programFile.Write(bytesBuffer.Bytes()); err != nil {
		fmt.Println(craft_error.Error{
			Msg:            fmt.Sprintf("[INTERNAL ERROR] [file a bug] failed to write the program: %s", err),
			RelativePath:   c.Context.RelativePath,
			GoFile:         c.Context.GoFile,
			MacroPosition:  craft_error.PositionFromToken(macro.MacroPosition),
			SourcePosition: craft_error.PositionFromToken(process.SourcePosition),
		}.Error())

		return
	}

	var goRunCmdOut bytes.Buffer

	goRunCmd := exec.Command("go", "run", ".")
	goRunCmd.Dir = dirPath
	goRunCmd.Stdout = &goRunCmdOut
	goRunCmd.Stderr = &goRunCmdOut

	if err := goRunCmd.Run(); err != nil {
		var msg string

		switch err.(type) {
		default:
			msg = fmt.Sprintf("[INTERNAL ERROR] [file a bug] failed to run the program: %s", err)
		case *exec.ExitError:
			msg = goRunCmdOut.String()
		}

		fmt.Println(craft_error.Error{
			Msg:            msg,
			RelativePath:   c.Context.RelativePath,
			GoFile:         c.Context.GoFile,
			MacroPosition:  craft_error.PositionFromToken(macro.MacroPosition),
			SourcePosition: craft_error.PositionFromToken(process.SourcePosition),
		}.Error())

		return
	}
}

func (c *Craft) addProcess(process Process) {
	c.processesMu.Lock()
	defer c.processesMu.Unlock()

	c.Processes = append(c.Processes, process)
}

func (c *Craft) addError(err craft_error.Error) {
	c.errsMu.Lock()
	defer c.errsMu.Unlock()

	c.Errs = append(c.Errs, err)
}

// DEBUG: remove this
// TODO: compare the size of a string source code vs an ast.File object
func _() {
	var (
		fileSet *token.FileSet
		astFile *ast.File
	)

	if err := printer.Fprint(os.Stdout, fileSet, astFile); err != nil {
		fmt.Printf("failed to printer Fprint: %s\n", err.Error())
		os.Exit(1)
		return
	}
}
