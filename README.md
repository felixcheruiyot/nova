# Nova

AI-first programming language — simple as Go, more efficient than Python, safer like Rust.

Read the [Design Specs](CLAUDE.md) for the full language specification.

---

## Build

**Requirements:** Go 1.21+

```bash
git clone https://github.com/your-org/nova
cd nova
go build -o nova .
```

Install to PATH (requires write access to `/usr/local/bin`):

```bash
sudo cp nova /usr/local/bin/nova
```

Verify:

```bash
nova version
# Nova 0.1-alpha
```

---

## Usage

### Run a file

```bash
nova run app.nv
```

### Start the REPL

```bash
nova shell
```

---

## Examples

### Hello World

```nova
print("Hello, World!")
```

```bash
nova run hello.nv
# Hello, World!
```

---

### Variables

```nova
name = "Alex"
age = 20
pi: float = 3.14
```

---

### Functions

```nova
func greet(name):
    return "Hello " + name

print(greet("Nova"))
```

```bash
# Hello Nova
```

---

### If / Else

```nova
age = 20

if age >= 18:
    print("Adult")
else:
    print("Minor")
```

```bash
# Adult
```

---

### For Loop

```nova
for i in 1..5:
    print(i)
```

```bash
# 1
# 2
# 3
# 4
# 5
```

---

### Lists and Maps

```nova
items = [1, 2, 3]
print(len(items))

user = {
    "name": "Alex",
    "age": 20
}
print(user["name"])
```

---

### String Interpolation

```nova
name = "Nova"
print("Hello {name}!")
```

```bash
# Hello Nova!
```

---

### Error Handling

```nova
try:
    result = 10 / 0
catch err:
    print("Error: {err}")
```

---

### Importing Modules

```nova
import math

print(math.pi)
print(math.sqrt(16))
```

```nova
from math import pi
print(pi)
```

---

### Full Example

```nova
func add(a, b):
    return a + b

result = add(5, 7)
print(result)
```

```bash
nova run app.nv
# 12
```

---

## REPL

```bash
nova shell
```

```
Nova REPL v0.1-alpha
Type 'exit' to quit

>> x = 5
>> x + 2
>> print(x)
5
>> exit
```

---

## Available Built-ins

| Function | Description |
|----------|-------------|
| `print(v)` | Print a value |
| `len(v)` | Length of string, list, or map |
| `type(v)` | Type name as string |
| `str(v)` | Convert to string |
| `int(v)` | Convert to integer |
| `float(v)` | Convert to float |
| `append(list, v)` | Append to list |
| `pop(list)` | Remove last element |
| `keys(map)` | Map keys as list |
| `values(map)` | Map values as list |
| `has(map, key)` | Check key exists |
| `upper(s)` | Uppercase string |
| `lower(s)` | Lowercase string |
| `split(s, sep)` | Split string |
| `join(list, sep)` | Join list to string |
| `contains(s, sub)` | Substring check |
| `trim(s)` | Trim whitespace |
| `abs(n)` | Absolute value |
| `sqrt(n)` | Square root |
| `pow(base, exp)` | Power |
| `floor(n)` | Floor |
| `ceil(n)` | Ceiling |
| `round(n)` | Round |
| `max(a, b)` | Maximum |
| `min(a, b)` | Minimum |

## Standard Modules

| Module | Contents |
|--------|----------|
| `math` | `pi`, `e`, `sqrt`, `sin`, `cos`, `log`, `pow`, and more |
| `io` | `read`, `write`, `print` |
| `json` | `encode`, `decode` |
| `time` | `now`, `sleep`, `format` |
| `fs` | `read`, `write`, `exists`, `delete` |

---

## File Extension

Nova source files use the `.nv` extension:

```
main.nv
server.nv
math_utils.nv
```

---

## Status

Nova is currently in **alpha (0.1)**. The interpreter is functional. The compiler and bytecode VM are planned for a future release.
