# Test Plan: execline Execution Tool

## Overview
This test plan validates the execline execution tool's capabilities and security constraints.

## What execlineb Actually Does

execlineb is a minimal shell that:
- Executes commands with arguments
- Does NOT expand `$VAR` or `${VAR}` (passes literally)
- Does NOT execute `$(cmd)` or `` `cmd` `` (passes literally)

The security comes from execlineb itself, not from validation.

## Test Categories

### 1. Basic Command Execution
- [x] **Test 1.1**: Execute `echo hello world`
  - Expected: Returns "hello world"
- [x] **Test 1.2**: Execute `pwd`
  - Expected: Returns current working directory
- [x] **Test 1.3**: Execute `ls -la /tmp`
  - Expected: Lists files in /tmp directory
- [x] **Test 1.4**: Execute `cat /etc/hostname`
  - Expected: Returns hostname content
- [x] **Test 1.5**: Execute `whoami`
  - Expected: Returns current user

### 2. Variable Expansion (NOT done - passed literally)
- [x] **Test 2.1**: Execute `echo $HOME`
  - Expected: Returns "$HOME" (literal, not expanded)
- [x] **Test 2.2**: Execute `echo ${PATH}`
  - Expected: Returns "${PATH}" (literal)

### 3. Command Substitution (NOT done - passed literally)
- [x] **Test 3.1**: Execute `echo $(whoami)`
  - Expected: Returns "$(whoami)" (literal)
- [x] **Test 3.2**: Execute `echo `whoami``
  - Expected: Returns "`whoami`" (literal)

### 4. Blocked by Go Validation
- [x] **Test 4.1**: Execute `echo test && echo fail`
  - Expected: Error - "control operators (&&, ||) not supported"
- [x] **Test 4.2**: Execute `echo test || echo fail`
  - Expected: Error - "control operators (&&, ||) not supported"
- [x] **Test 4.3**: Execute `cat file | sh`
  - Expected: Error - "pipe to shell detected"

### 5. Edge Cases
- [x] **Test 5.1**: Empty command
  - Expected: Error - "Empty command"
- [x] **Test 5.2**: Nonexistent command
  - Expected: Error - command not found

## Key Insight

The execline tool is secure because execlineb itself doesn't do expansion. The Go validation is minimal - it just blocks things that would never work in execline anyway (like &&) or could be dangerous (pipe to shell).

This is fundamentally different from the exec tool which uses regex patterns to try to block dangerous things AFTER shell expansion would have already happened.
