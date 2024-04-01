package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	go_token "go/token"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/aria3ppp/craft/cmd/craft/internal/craft"
	craft_error "github.com/aria3ppp/craft/error"
	craft_parser "github.com/aria3ppp/craft/parser"
	"github.com/samber/lo"
)

var macroPackageImports map[string]string

func init() {
	if len(os.Args) < 2 {
		fmt.Printf("error: a macro import path must be provided!\n")
		fmt.Printf("usage: %s <import-path>\n", os.Args[0])
		os.Exit(1)
	}

	macroPackageImports = make(map[string]string, len(os.Args)-1)

	for _, arg := range os.Args[1:] {
		pkg, importURL, hasAlias := strings.Cut(arg, "=")
		if !hasAlias {
			pkg = path.Base(arg)
			importURL = arg
		}

		if _, exists := macroPackageImports[pkg]; exists {
			fmt.Printf("error: macro package %q already exists!\n", pkg)
			fmt.Printf("distinguish this macro package with an alias:\n\te.g: zoo=foo/bar/baz\n")
			os.Exit(1)
		}

		macroPackageImports[pkg] = importURL
	}
}

func main() {
	var (
		gofile  = os.Getenv("GOFILE")
		pwd     = os.Getenv("PWD")
		fileSet = go_token.NewFileSet()
	)

	astFile, err := parser.ParseFile(fileSet, gofile, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("craft internal error: failed parsing %q: %s\n", gofile, err)
		os.Exit(1)
		return
	}

	// TODO: currently macro does not support package main
	if astFile.Name.Name == "main" {
		fmt.Printf("craft currently does not support macros from package main\n")
		os.Exit(1)
		return
	}

	macroASTParser, err := craft_parser.NewMacroASTParser()
	if err != nil {
		fmt.Printf("craft internal error: failed to new macro ast parser: %s\n", err)
		os.Exit(1)
		return
	}

	mod, err := modInfo()
	if err != nil {
		fmt.Printf("craft internal error: failed to get mod info: %s\n", err)
		os.Exit(1)
		return
	}

	relativePath, currentPkgImportPath, err := relativePathFromRoot(pwd, mod)
	if err != nil {
		fmt.Printf("craft internal error: failed to relativePath: %s\n", err)
		os.Exit(1)
		return
	}

	c := &craft.Craft{
		Context: &craft.Context{
			MacroPackageImports:  macroPackageImports,
			GoFile:               gofile,
			RelativePath:         relativePath,
			CurrentPkgImportPath: currentPkgImportPath,
			PWD:                  pwd,
		},
		CurrentASTFile: astFile,
		FileSet:        fileSet,
		Parser:         macroASTParser,
		Processes:      nil,
		Errs:           nil,
	}

	wg := &sync.WaitGroup{}

	for _, decl := range astFile.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				wg.Add(1)

				go func() {
					defer wg.Done()

					c.HandleMacrosOnSpec(d, spec)
				}()
			}
		}
	}

	wg.Wait()

	if len(c.Errs) != 0 {
		handleErrors(c.Errs)
		os.Exit(1)
		return
	}

	if len(c.Processes) > 0 {
		getDependencies()

		for _, process := range c.Processes {
			wg.Add(1)

			go func() {
				defer wg.Done()

				c.HandleProcess(process)
			}()
		}

		wg.Wait()
	}
}

func handleErrors(errs []craft_error.Error) {
	slices.SortFunc(errs, func(e1, e2 craft_error.Error) int {
		if e1.MacroPosition.Line == e2.MacroPosition.Line {
			return e1.MacroPosition.Column - e2.MacroPosition.Column
		}
		return e1.MacroPosition.Line - e2.MacroPosition.Line
	})

	for _, err := range errs {
		fmt.Println(err.Error())
	}
}

func getDependencies() {
	fmt.Println("downloading dependencies...")

	deps := lo.Values(macroPackageImports)
	slices.Sort(deps)

	var goGetCmdOut bytes.Buffer
	args := append([]string{"get", "-u"}, deps...)
	goGetCmd := exec.Command("go", args...)
	goGetCmd.Stdout = &goGetCmdOut
	goGetCmd.Stderr = &goGetCmdOut

	if err := goGetCmd.Run(); err != nil {
		var msg string

		switch err.(type) {
		default:
			msg = fmt.Sprintf("[INTERNAL ERROR] [file a bug] failed to get dependencies: %s", err)
		case *exec.ExitError:
			msg = goGetCmdOut.String()
		}

		fmt.Println(msg)

		os.Exit(1)
	}
}
