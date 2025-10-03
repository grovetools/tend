# Automatic PATH Handling for Mock Binaries

This example demonstrates the automatic PATH handling feature for TUI sessions with mock binaries.

**NOTE**: This is primarily a code documentation example showing the API usage. The framework feature is fully implemented and working as demonstrated in the tend framework's own tests.

## What This Example Shows

When testing applications that call external commands, you often need to provide mock versions of those commands. Previously, this required:

1. Creating a wrapper script that manually sets `PATH`
2. Making the wrapper executable
3. Launching the wrapper instead of your actual binary

**Now**, the `tend` framework handles this automatically!

## How It Works

The `harness.Context.StartTUI` function automatically detects when you've set up mock binaries using the `test_bin_dir` context key. When found, it:

1. Prepends your mock directory to the `PATH` environment variable
2. Passes the modified `PATH` to the TUI subprocess
3. Ensures your mocks are found first, before system binaries

## Key Convention

```go
// Set your mock binary directory using this specific key:
ctx.Set("test_bin_dir", mockDir)

// Then just launch your TUI normally - PATH is handled automatically!
session, err := ctx.StartTUI(binaryPath, args)
```

## Running This Example

```bash
# Build the example
go build -o bin/auto-path-mocks ./examples/auto-path-mocks

# Run it
./bin/auto-path-mocks run auto-path-mocks
```

## What You'll See

The example creates mock versions of `git` and `curl`, then runs a script that calls these commands. You'll see output like:

```
MOCK GIT: This is a mock git binary!
MOCK CURL: Pretending to fetch https://example.com
```

This proves that the mocks are being called instead of the real system binaries, thanks to automatic PATH handling.

## Comparison: Old vs New Approach

### Old Approach (Manual Wrapper)

```go
// Create a wrapper script
wrapperContent := fmt.Sprintf(`#!/bin/bash
export PATH="%s:$PATH"
exec "%s" "$@"
`, mockDir, actualBinary)

wrapperPath := filepath.Join(ctx.RootDir, "wrapper")
fs.WriteString(wrapperPath, wrapperContent)
os.Chmod(wrapperPath, 0755)

// Launch the wrapper
session, err := ctx.StartTUI(wrapperPath, "arg1", "arg2")
```

### New Approach (Automatic)

```go
// Just set the convention key
ctx.Set("test_bin_dir", mockDir)

// Launch normally - PATH is automatic!
session, err := ctx.StartTUI(actualBinary, []string{"arg1", "arg2"})
```

## Benefits

- **Cleaner code**: No wrapper script generation
- **Less error-prone**: No file permissions to manage
- **More maintainable**: Standard convention across all tests
- **Automatic**: Works consistently with `StartTUI` and `Command`
