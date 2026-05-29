package main

import (
	"bufio"
	"fmt"
	"nova/ast"
	"nova/lexer"
	"nova/parser"
	"nova/pkgmgr"
	"nova/runtime"
	"nova/vm"
	"os"
	"strings"
)

func main() {
	// Wire the file-module loader so runtime can execute user packages without
	// importing lexer/parser (which would create an import cycle).
	runtime.FileRunner = func(src string, env *runtime.Environment) error {
		return executeInEnv(src, env)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: nova run <file.nv|file.nvc>")
			os.Exit(1)
		}
		runFile(os.Args[2])

	case "build":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: nova build <file.nv>")
			os.Exit(1)
		}
		buildFile(os.Args[2])

	case "shell":
		runREPL()

	case "add":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: nova add <package>")
			os.Exit(1)
		}
		if err := pkgmgr.Add(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}

	case "remove":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: nova remove <package>")
			os.Exit(1)
		}
		if err := pkgmgr.Remove(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}

	case "list":
		if err := pkgmgr.List(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}

	case "version":
		fmt.Println("Nova 0.1-alpha")

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`Nova Programming Language 0.1-alpha

Usage:
  nova run <file.nv>      Run a Nova source file (interpreter)
  nova run <file.nvc>     Run compiled Nova bytecode (VM)
  nova build <file.nv>    Compile to bytecode (.nvc)
  nova shell              Start the interactive REPL
  nova add <package>      Add a package dependency
  nova remove <package>   Remove a package dependency
  nova list               List declared dependencies
  nova version            Print version info
`)
}

// ── Parse ─────────────────────────────────────────────────────────────────────

func parseSrc(src string) (*ast.Program, error) {
	l := lexer.New(src)
	tokens, err := l.Tokenize()
	if err != nil {
		return nil, err
	}
	return parser.New(tokens).Parse()
}

// ── Run (interpreter) ─────────────────────────────────────────────────────────

func runFile(path string) {
	if strings.HasSuffix(path, ".nvc") {
		runBytecode(path)
		return
	}
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	prog, err := parseSrc(string(src))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := runtime.NewInterpreter().Run(prog); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// ── Build (compiler) ──────────────────────────────────────────────────────────

func buildFile(path string) {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	prog, err := parseSrc(string(src))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	proto, err := vm.CompileProgram(prog)
	if err != nil {
		fmt.Fprintln(os.Stderr, "compile error:", err)
		os.Exit(1)
	}
	out := strings.TrimSuffix(path, ".nv") + ".nvc"
	f, err := os.Create(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := vm.Encode(f, proto); err != nil {
		fmt.Fprintln(os.Stderr, "error writing bytecode:", err)
		os.Exit(1)
	}
	fmt.Printf("Compiled %s → %s\n", path, out)
}

// ── Run (VM) ──────────────────────────────────────────────────────────────────

func runBytecode(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer f.Close()
	proto, err := vm.Decode(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading bytecode:", err)
		os.Exit(1)
	}
	machine := vm.New()
	if err := machine.RunProto(proto); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// ── REPL ──────────────────────────────────────────────────────────────────────

func runREPL() {
	fmt.Println("Nova REPL v0.1-alpha")
	fmt.Println("Type 'exit' to quit, '?name' to inspect a variable\n")

	interp := runtime.NewInterpreter()
	scanner := bufio.NewScanner(os.Stdin)
	var buf strings.Builder
	indent := 0

	for {
		if buf.Len() == 0 {
			fmt.Print(">> ")
			indent = 0
		} else {
			fmt.Print(".. ")
		}

		if !scanner.Scan() {
			break
		}
		line := scanner.Text()

		trimmed := strings.TrimSpace(line)
		if trimmed == "exit" {
			break
		}

		// Variable inspection: ?varname
		if strings.HasPrefix(trimmed, "?") {
			name := strings.TrimSpace(trimmed[1:])
			src := fmt.Sprintf("print(%s)", name)
			prog, err := parseSrc(src)
			if err == nil {
				_ = interp.Run(prog)
			}
			continue
		}

		buf.WriteString(line)
		buf.WriteString("\n")

		// Track block depth: lines ending with ':' open a block
		if strings.HasSuffix(trimmed, ":") {
			indent++
			continue
		}
		// Empty line closes one block level when indented
		if indent > 0 && trimmed == "" {
			indent--
			if indent > 0 {
				continue
			}
		}
		if indent > 0 {
			continue
		}

		src := buf.String()
		buf.Reset()
		prog, err := parseSrc(src)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			continue
		}
		if err := interp.Run(prog); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
	}
	fmt.Println("\nBye!")
}

// ── Shared helpers ────────────────────────────────────────────────────────────

// executeInEnv parses src and runs it inside env — used by the file-module loader.
func executeInEnv(src string, env *runtime.Environment) error {
	prog, err := parseSrc(src)
	if err != nil {
		return err
	}
	return runtime.NewInterpreterWithEnv(env).Run(prog)
}
