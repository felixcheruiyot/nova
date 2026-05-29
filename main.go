package main

import (
	"bufio"
	"fmt"
	"nova/lexer"
	"nova/parser"
	"nova/runtime"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: nova run <file.nv>")
			os.Exit(1)
		}
		runFile(os.Args[2])
	case "shell":
		runREPL()
	case "version":
		fmt.Println("Nova 0.1-alpha")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Nova Programming Language 0.1-alpha

Usage:
  nova run <file.nv>    Run a Nova source file
  nova shell            Start the interactive REPL
  nova version          Print version info`)
}

func runFile(path string) {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := execute(string(src)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func execute(src string) error {
	l := lexer.New(src)
	tokens, err := l.Tokenize()
	if err != nil {
		return err
	}
	p := parser.New(tokens)
	prog, err := p.Parse()
	if err != nil {
		return err
	}
	interp := runtime.NewInterpreter()
	return interp.Run(prog)
}

func runREPL() {
	fmt.Println("Nova REPL v0.1-alpha")
	fmt.Println("Type 'exit' to quit\n")

	interp := runtime.NewInterpreter()
	scanner := bufio.NewScanner(os.Stdin)
	var buf strings.Builder

	for {
		if buf.Len() == 0 {
			fmt.Print(">> ")
		} else {
			fmt.Print(".. ")
		}

		if !scanner.Scan() {
			break
		}
		line := scanner.Text()

		if strings.TrimSpace(line) == "exit" {
			break
		}

		// Multi-line: if line ends with : accumulate
		buf.WriteString(line)
		buf.WriteString("\n")

		// Heuristic: if not inside a block, execute
		src := buf.String()
		if !strings.HasSuffix(strings.TrimRight(src, "\n"), ":") {
			l := lexer.New(src)
			tokens, err := l.Tokenize()
			if err != nil {
				fmt.Fprintln(os.Stderr, "lexer error:", err)
				buf.Reset()
				continue
			}
			p := parser.New(tokens)
			prog, err := p.Parse()
			if err != nil {
				fmt.Fprintln(os.Stderr, "parse error:", err)
				buf.Reset()
				continue
			}
			if err := interp.Run(prog); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
			}
			buf.Reset()
		}
	}
	fmt.Println("\nBye!")
}
