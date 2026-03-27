# execline-hardening Skill

Security hardening for agentic systems using execline instead of bash.

## What is execline?

execline is a minimal scripting language from [skarnet.org](https://skarnet.org/software/execline/) designed for security and simplicity. It uses **chain loading** - each command execs into the next one, rather than staying resident like a shell.

## Installation

```bash
# Debian/Ubuntu/Armbian
apt install execline

# Alpine
apk add execline
```

## Key Concepts

### Chain Loading
execline uses exec() heavily - each program runs, then execs into the next one:
```
execlineb -c "nice -10 echo hello"
```
This is more efficient than spawning a shell interpreter.

### Whitespace is Whitespace
Newlines, spaces, and tabs are all treated the same - they're just word separators.

### Blocks
Curly braces `{ }` create blocks to group commands with their arguments:
```
foreground { echo hello } echo world
```

## Security Properties

### What execline DOESN'T do:
- **Command substitution**: `$(cmd)`, `` `cmd` `` - passed literally, not executed
- **Shell control operators**: `&&`, `||`, `;` - these are just arguments
- **Pipes to shell**: `| sh`, `| bash` - not supported

### What execline DOES do (differently from sh):
- **Variable substitution**: Uses a deliberate substitution mechanism, not shell-style `$VAR`
- This is a feature, not a bug - it provides predictable behavior

## Variable Management

execline has a deliberate variable system - no shell-style `$VAR` magic:

### define - Define a literal substitution
```bash
define FOO hello
echo $FOO
# Output: hello
```

### importas - Import environment variable
```bash
importas home HOME
cd $home
ls
```

### backtick - Command output to variable (with -E flag!)
```bash
backtick DATE { date +%Y-%m-%d }
echo $DATE
# Output: today's date
```

**Note**: Unlike shell's `$(date)`, execline uses `backtick` which:
- Runs the command
- Captures stdout
- Stores it in an environment variable
- Then execs into the next command

### backtick -E - Auto-import command substitution
The `-E` flag makes backtick automatically import the result as a variable, enabling true command substitution (like `$()` in bash):

```bash
backtick -E DATE { date } echo $DATE
# Output: Sun Mar 22 08:08:58 PM GMT 2026
```

Without `-E`, you need `importas` to access the variable. With `-E`, it's auto-imported directly.

## Sequencing Commands

### foreground - Run and wait
```bash
foreground { echo first } echo second
# Output:
# first
# second
```

### background - Run in background
```bash
background { long-running-task } echo done
# Starts task, immediately prints "done"
```

## Conditionals

### if - Run if condition succeeds
```bash
if { test -f /etc/passwd } echo file exists
```

### if with negation
```bash
if -n { test -f /tmp/test } mkdir /tmp/test
# -n negates: if file DOESN'T exist, create it
```

### ifelse - If-else
```bash
ifelse { test -d $HOME }
{ echo "It's a directory" }
{ echo "Not a directory" }
```

## Loops

### forx - Iterate over list
```bash
forx item { alpha beta gamma } echo $item
# Output: alpha, beta, gamma (each on separate line via foreground)
```

### forstdin - Read from stdin
```bash
echo -e "a\nb\nc" | forstdin line echo $line
```

## File Operations

### elglob - File globbing
```bash
elglob files /etc/f* echo ${files}
# Lists all files in /etc starting with 'f'
```

### redirfd - Redirect file descriptors
```bash
redirfd -w 1 output.txt echo hello
# Redirect stdout (fd 1) to file

redirfd -a 1 log.txt date
# Append to log

redirfd -r 0 /dev/null cat
# Redirect stdin from /dev/null
```

### fdmove - Move file descriptors
```bash
fdmove -c 2 1 prog
# Duplicate fd 2 (stderr) to fd 1 (stdout) - stderr to stdout
```

## Example Scripts

### Simple sequence
```bash
#!/bin/execlineb -P
importas home HOME
cd $home
ls
```

### Conditional file creation
```bash
#!/bin/execlineb -P
importas home HOME
if -n { test -d ${home}/.cache }
mkdir -p ${home}/.cache
echo "Cache directory ready"
```

### Loop and create files
```bash
#!/bin/execlineb -P
forx name { alpha beta gamma }
{
  touch /tmp/${name}
}
echo "Files created"
```

### Pipeline
```bash
#!/bin/execlineb -P
pipeline { ls /etc }
wc -l
# Count files in /etc
```

## Comparison: execline vs shell (sh/bash)

| Feature | sh/bash | execline |
|---------|---------|----------|
| $VAR expansion | Yes | Yes (via substitution) |
| ${VAR} expansion | Yes | Yes |
| $(cmd) substitution | Yes | **Yes** - use `backtick -E VAR { cmd }` |
| `cmd` substitution | Yes | **No** (use backtick) |
| &&, \|\| | Yes | **No** - use if/foreground |
| ; | Yes | **No** - use foreground |
| Variable assignment | VAR=value | define VAR value |
| Command output | $(cmd) | backtick VAR { cmd } |
| Loops | for, while | forx, forstdin |
| Conditionals | if/then/else | if, ifelse |

## Usage in picoclaw

The picoclaw agent has a built-in `execline` tool that you can call directly:

```
Tool: execline
ToolInput: { "command": "define FOO bar echo $FOO" }
```
→ Output: bar

The ExeclineTool validates:
- No `&&`, `||` (use `if`, `foreground` instead)
- No pipes to shell (`| sh`, `| bash`)

## Testing

```bash
# Variable substitution works:
execlineb -c 'define FOO bar echo $FOO'
# Output: bar

# Environment variables:
HOME=/tmp execlineb -c 'importas h HOME cd $h pwd'
# Output: /tmp

# backtick -E enables command substitution (like $()):
execlineb -c 'backtick -E DATE { date } echo $DATE'
# Output: current date/time

# backtick without -E requires importas:
execlineb -c 'backtick DATE { date } importas D DATE echo $D'
# Output: current date/time
```

## Recommendations

1. **Use execline** for scripts that don't need `$(cmd)` or `&&`/`||`
2. Use `if` instead of `&&`, `ifelse` instead of `if-then-else`
3. Use `backtick` instead of `$(...)`
4. Use `foreground` for sequential commands

## When Exec Tool is Unavailable

If the `exec` tool is disabled but you still need command execution:

1. **Generate execline commands** for the user to execute manually
2. **Write scripts** that the user can save and run

Example - Creating a script for the user:

```bash
#!/usr/bin/execlineb -S0
cd /home/infra
export HOME /home/infra
foreground { echo "Environment configured" }
```

The user saves this to a file and runs it. This gives you a way to help even without direct execution capabilities.

## Positional Arguments

execline scripts can handle command-line arguments using `-sN` flag:

```bash
#!/usr/bin/execlineb -s0
# $1 is first arg, $@ is all remaining args
echo "First: $1"
echo "All: $@"
```

- `-s0`: $1, $2, etc. work directly
- `-s1`: Shift after first argument
- `-sN`: Positionalize N arguments

This is useful for wrapper scripts that delegate to other commands.
