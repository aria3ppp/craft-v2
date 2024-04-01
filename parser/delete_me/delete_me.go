package main

import (
	"encoding/json"
	"fmt"
	"os"

	craft_parser "github.com/aria3ppp/craft/parser"
)

func main() {
	macroInvocationParser, err := craft_parser.NewMacroASTParser()
	if err != nil {
		fmt.Printf("craft internal error: failed to new macro invocation parser: %s\n", err)
		os.Exit(1)
		return
	}

	ss := []string{
		"",
		"x",
		" #",
		" x #",
		" # x",
		" # x .",
		" # x . x",
		" # x . x x",
		" # x . x ()",
		" # x . x ( x )",
		" # x . x ( 'x' )",
		" # x . x ( `x` )",
		" # x . x ( `x` x )",
		" # x . x ( `x` ) x",
		" #macro.macro",
	}

	for _, s := range ss {
		mi, err := macroInvocationParser.ParseString("", s)
		fmt.Println("input:", s)
		if err != nil {
			fmt.Println("error:", err.Error())
		}
		j, err := json.Marshal(mi)
		if err != nil {
			panic(err)
		}
		fmt.Println("ast:", string(j))
		fmt.Println()
	}
}
