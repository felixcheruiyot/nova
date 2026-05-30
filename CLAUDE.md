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

Built-in modules available at runtime (no install required):

```text
io      — print, read, write
math    — sqrt, pow, abs, floor, ceil, round, log, sin, cos, pi, e, inf, max, min
json    — stringify, parse
time    — now, sleep, format
fs      — read, write, exists, lines
ai      — prompt, summarize, translate, classify, extract
```

Note: `http` is not an importable module. HTTP servers are declared with the built-in `server { }` DSL.

---

# Module System

## Importing a built-in module

```nova
import math
print(math.sqrt(16))
print(math.pi)
```

After `import math`, the module is bound to the name `math` in the current scope.

## Selective import

```nova
from json import stringify, parse

data = {"name": "Alex"}
print(stringify(data))
```

Named exports are bound directly into scope — no `json.` prefix needed.

## User-defined modules

Place `.nv` files inside a `nova_modules/` directory next to your script.
The file's top-level variables and functions become the module's exports.

```text
nova_modules/
  utils.nv
```

```nova
# nova_modules/utils.nv
func double(x):
    return x * 2
```

```nova
# main.nv
import utils
print(utils.double(5))   # 10

from utils import double
print(double(7))         # 14
```

## Module reference

### io

```nova
import io
io.print("hello")
line = io.read()
io.write("no newline")
```

### math

```nova
import math
math.sqrt(9)        # 3
math.pow(2, 8)      # 256
math.abs(-5)        # 5
math.floor(3.9)     # 3
math.ceil(3.1)      # 4
math.round(3.5)     # 4
math.max(1, 2)      # 2
math.min(1, 2)      # 1
math.log(math.e)    # 1
math.sin(0)         # 0
math.cos(0)         # 1
math.pi             # 3.14159…
math.e              # 2.71828…
```

### json

```nova
import json
s = json.stringify({"a": 1})   # '{"a":1}'
v = json.parse('{"a":1}')      # map
```

### time

```nova
import time
ms = time.now()              # Unix milliseconds
time.sleep(500)              # sleep 500 ms
s  = time.format("15:04:05") # formatted current time
```

### fs

```nova
import fs
text  = fs.read("data.txt")
fs.write("out.txt", "hello")
ok    = fs.exists("data.txt")
lines = fs.lines("data.txt")   # list of strings
```

### ai

Requires the `ANTHROPIC_API_KEY` environment variable.

```nova
import ai
reply     = ai.prompt("What is Nova?")
summary   = ai.summarize("Long text here…")
fr        = ai.translate("Hello", "French")
category  = ai.classify("I love this!", ["positive", "negative"])
fields    = ai.extract("John is 30 years old", "name, age")
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
