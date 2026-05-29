# CLAUDE.md

# Nova Programming Language Specification

Version: 0.1-alpha

Nova is a modern interpreted + optionally compiled programming language inspired by Python, Go, and TypeScript.

The goal of Nova is to provide:

* Python simplicity
* Fast execution
* Modern tooling
* AI-native capabilities
* Safe developer ergonomics
* Beginner-friendly syntax

---

# Core Philosophy

Nova should feel:

* readable
* minimal
* expressive
* safe
* predictable

The language prioritizes developer experience over academic purity.

---

# File Extensions

```text
.nv
```

Examples:

```text
main.nv
server.nv
math.nv
```

---

# Runtime Modes

Nova supports two modes:

1. Interpreted Mode
2. Compiled Mode

Interpreter command:

```bash
nova run app.nv
```

Compiler command:

```bash
nova build app.nv
```

---

# Language Characteristics

| Feature           | Value                                    |
| ----------------- | ---------------------------------------- |
| Typing            | Dynamic with optional static annotations |
| Memory Management | Automatic garbage collection             |
| Concurrency       | Lightweight tasks                        |
| Syntax Style      | Python-inspired                          |
| Blocks            | Indentation OR braces                    |
| Runtime           | VM-based                                 |
| Package Manager   | Built-in                                 |
| Null Safety       | Optional chaining                        |
| AI Integration    | Native                                   |

---

# Interpreter Architecture

The Nova interpreter consists of:

```text
Source Code
    ↓
Tokenizer / Lexer
    ↓
Parser
    ↓
AST (Abstract Syntax Tree)
    ↓
Semantic Analyzer
    ↓
Bytecode Generator
    ↓
Nova Virtual Machine
```

---

# Directory Structure

```text
nova/
├── lexer/
├── parser/
├── ast/
├── runtime/
├── vm/
├── std/
├── cli/
├── compiler/
└── tests/
```

---

# Lexer Specification

The lexer converts raw source code into tokens.

Supported token categories:

```text
IDENTIFIER
NUMBER
STRING
KEYWORD
OPERATOR
NEWLINE
INDENT
DEDENT
EOF
```

Example:

Nova source:

```nova
name = "Alex"
```

Tokens:

```text
IDENTIFIER(name)
EQUALS
STRING("Alex")
```

---

# Reserved Keywords

```text
if
else
for
while
func
return
break
continue
task
wait
import
from
class
true
false
null
server
get
post
put
delete
```

---

# Primitive Types

```text
int
float
string
bool
list
map
null
```

Future:

```text
set
tuple
bytes
decimal
```

---

# Variables

Variables are declared automatically.

Example:

```nova
age = 15
```

Optional typing:

```nova
age: int = 15
```

---

# Functions

Syntax:

```nova
func greet(name):
    return "Hello " + name
```

AST representation:

```text
FunctionNode
  name: greet
  params: [name]
  body: [...]
```

---

# Control Flow

## If Statements

```nova
if age >= 18:
    print("Adult")
else:
    print("Minor")
```

## Loops

```nova
for i in 1..5:
    print(i)
```

---

# Expressions

Supported operators:

```text
+
-
*
/
%
==
!=
>
<
>=
<=
&&
||
!
```

---

# Optional Chaining

Syntax:

```nova
user?.profile?.email
```

If any value is null:

```text
returns null
```

instead of crashing.

---

# Collections

## Lists

```nova
items = [1,2,3]
```

## Maps

```nova
user = {
    "name": "Alex",
    "age": 15
}
```

---

# String Interpolation

Syntax:

```nova
print("Hello {name}")
```

Implementation should evaluate expressions inside braces.

---

# Standard Library

Initial built-in modules:

```text
io
math
http
json
time
fs
ai
```

---

# AI Module

Native AI functions:

```nova
summary = ai.summarize(text)
translation = ai.translate(text, "fr")
```

The runtime should expose AI capabilities as first-class APIs.

---

# Concurrency Model

Nova uses lightweight tasks.

Syntax:

```nova
task fetchUsers()
task fetchPayments()

wait
```

Runtime requirements:

* cooperative scheduling
* lightweight fibers/coroutines
* shared event loop

---

# Error Handling

Syntax:

```nova
try:
    risky()

catch err:
    print(err)
```

Goals:

* readable errors
* exact line references
* helpful suggestions

Example:

```text
Line 14:
Expected closing bracket '}'
```

---

# Parser Rules

The parser should support:

* indentation parsing
* optional brace parsing
* Pratt parser for expressions
* recursive descent parsing

Recommended strategy:

```text
Recursive Descent Parser
+
Pratt Expression Parser
```

---

# AST Node Types

Core nodes:

```text
ProgramNode
FunctionNode
VariableNode
AssignmentNode
CallNode
IfNode
LoopNode
ReturnNode
BinaryExpressionNode
LiteralNode
```

---

# Bytecode VM

The VM executes Nova bytecode.

Example instructions:

```text
LOAD_CONST
LOAD_VAR
STORE_VAR
CALL
RETURN
ADD
SUB
JUMP
JUMP_IF_FALSE
```

Example:

Nova:

```nova
x = 5 + 2
```

Bytecode:

```text
LOAD_CONST 5
LOAD_CONST 2
ADD
STORE_VAR x
```

---

# Garbage Collection

Recommended:

```text
Mark-and-Sweep GC
```

Future optimization:

```text
Generational GC
```

---

# CLI Commands

## Run File

```bash
nova run app.nv
```

## Build

```bash
nova build app.nv
```

## Test

```bash
nova test
```

## Format

```bash
nova fmt
```

## REPL

```bash
nova shell
```

---

# REPL Requirements

Interactive shell should support:

* multiline input
* syntax highlighting
* command history
* variable inspection

Example:

```text
Nova REPL v0.1

>> x = 5
>> x + 2
7
```

---

# Package Manager

Built-in package manager called:

```text
nova.pm
```

Install:

```bash
nova add requests
```

Remove:

```bash
nova remove requests
```

---

# HTTP Server DSL

Built-in API framework.

Example:

```nova
server {

    get "/hello" {
        return {
            "message": "Hello"
        }
    }

}
```

---

# Performance Goals

Initial targets:

| Metric            | Goal                    |
| ----------------- | ----------------------- |
| Startup Time      | <50ms                   |
| Memory Usage      | Lower than Python       |
| Async Performance | Similar to Node.js      |
| Build Speed       | Fast incremental builds |

---

# Future Compiler Targets

Planned outputs:

```text
Native Binary
WebAssembly
LLVM IR
```

---

# Recommended Implementation Languages

Best options:

| Language | Why                  |
| -------- | -------------------- |
| Rust     | Safety + performance |
| Go       | Simplicity           |
| Zig      | Low-level control    |
| C++      | Mature ecosystem     |

Primary recommendation:

```text
Rust
```

---

# Example Full Program

```nova
func add(a, b):
    return a + b

result = add(5, 7)

print(result)
```

Expected output:

```text
12
```

---

# MVP Milestones

## Phase 1

* lexer
* parser
* interpreter
* variables
* functions
* loops
* conditions

## Phase 2

* modules
* package manager
* bytecode VM
* REPL

## Phase 3

* compiler
* async runtime
* AI runtime
* web framework

---

# Non-Goals

Nova should NOT initially support:

* macros
* operator overloading
* manual memory management
* complex metaprogramming
* inheritance-heavy OOP

---

# Philosophy Summary

Nova exists to make modern software engineering:

* simpler
* safer
* faster
* more enjoyable

without sacrificing power.
