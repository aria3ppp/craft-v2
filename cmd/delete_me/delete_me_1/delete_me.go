package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	go_token "go/token"
	"os"
	// internal_parser "github.com/aria3ppp/craft/cmd/craft/internal/parser"
)

func main() {
	var (
		gofile = os.Getenv("GOFILE")
		// pwd     = os.Getenv("PWD")
		fileSet = go_token.NewFileSet()
	)

	astFile, err := parser.ParseFile(fileSet, gofile, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("craft internal error: failed parsing %q: %s\n", gofile, err)
		os.Exit(1)
		return
	}

	// macroInvocationParser, err := internal_parser.NewMacroInvocationParser()
	// if err != nil {
	// 	fmt.Printf("craft internal error: failed to new macro invocation parser: %s\n", err)
	// 	os.Exit(1)
	// 	return
	// }

	for _, decl := range astFile.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {

			if genDecl.Doc != nil {
				fmt.Println("group comment {")
				for _, c := range genDecl.Doc.List {
					fmt.Println(c)
				}
				fmt.Println("} end of group comment")
			}

			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					fmt.Println(typeSpec)
					if typeSpec.Doc != nil {
						fmt.Print("doc: ")
						fmt.Println(typeSpec.Doc.List)
					}
					if typeSpec.Comment != nil {
						fmt.Print("line comment: ")
						fmt.Println(typeSpec.Comment.List)
					}
					fmt.Println()
				}
			}
		}
	}
}
