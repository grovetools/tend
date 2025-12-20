# TUI Testing Quick Reference

## Simplified API (Recommended)

### ✅ Use `Type()` for most interactions

The `Type()` method combines sending keys + waiting for stability. This is what you want 90% of the time:

```go
// Navigate down
session.Type("j")

// Go to top (vim-style)
session.Type("g", "g")

// Open search and type
session.Type("/")
session.Type("my-search")
session.Type("Enter")

// Toggle feature
session.Type("A")  // Automatically waits for tree rebuild
```

### When to use other methods

**Use `SendKeys()` + custom wait** when you need specific waiting behavior:

```go
// Wait for specific text to appear
session.SendKeys("/")
session.WaitForText("Search:", 3*time.Second)

// Wait for any of several texts
session.SendKeys("?")
session.WaitForAnyText([]string{"Help", "Usage"}, 5*time.Second)

// No wait (rare - only for final actions like quit)
session.SendKeys("q")  // Don't wait after quitting
```

## Common Patterns

### Navigation

```go
// Simple navigation
session.Type("j")  // Down
session.Type("k")  // Up
session.Type("h")  // Left/collapse
session.Type("l")  // Right/expand

// Jump to top/bottom
session.Type("g", "g")  // Top
session.Type("G")       // Bottom
```

### Search and Filter

```go
// Open search
session.Type("/")
session.WaitForText("Search:", 2*time.Second)

// Type search term
session.Type("my-term")

// Execute search
session.Type("Enter")
session.WaitForText("my-term", 3*time.Second)

// Clear search
session.Type("Escape")
```

### Assertions

```go
// Check for text
session.AssertContains("expected text")
session.AssertNotContains("unexpected text")

// Check for line matching pattern
session.AssertLine(func(line string) bool {
    return strings.HasPrefix(line, ">") && strings.Contains(line, "focused")
}, "Expected focused item")
```

### Toggle Features

```go
// Toggle and wait for tree rebuild
session.Type("A")  // Archives
session.AssertContains(".archive")

// Toggle off
session.Type("A")
session.AssertNotContains(".archive")
```

## Migration from Old API

### Before (verbose)

```go
if err := session.SendKeys("j"); err != nil {
    return err
}
if err := session.WaitStable(); err != nil {
    return err
}
```

### After (concise)

```go
if err := session.Type("j"); err != nil {
    return err
}
```

Or even simpler in a test step:

```go
session.Type("j")  // Errors propagate automatically in test context
```

## Anti-Patterns to Avoid

### ❌ Don't use `SendKeysAndWaitForChange`

```go
// AVOID - this times out if screen doesn't change
session.SendKeysAndWaitForChange(2*time.Second, "g", "g")
```

Use `Type()` instead - it waits for stability, not just change:

```go
// CORRECT
session.Type("g", "g")
```

### ❌ Don't use bare `SendKeys` for navigation

```go
// AVOID - doesn't wait for UI to update
session.SendKeys("j")
// Next line might execute before UI updates!
session.AssertContains("item-2")
```

Use `Type()` which waits automatically:

```go
// CORRECT
session.Type("j")
session.AssertContains("item-2")  // UI is already stable
```

### ❌ Don't add arbitrary sleeps

```go
// AVOID - arbitrary timing
session.SendKeys("A")
time.Sleep(2 * time.Second)
```

Use `Type()` which waits for actual stability:

```go
// CORRECT
session.Type("A")  // Waits until UI is actually stable
```

## Decision Tree

```
Do you need to send keys?
├─ YES
│   ├─ Do you know what text will appear?
│   │   ├─ YES → SendKeys() + WaitForText("expected")
│   │   └─ NO  → Type() (waits for stability)
│   │
│   └─ Is this the final action (like quit)?
│       ├─ YES → SendKeys() (no wait needed)
│       └─ NO  → Type() (default choice)
│
└─ NO (just asserting/capturing)
    └─ Use AssertContains, AssertLine, or Capture
```

## Summary

**Default choice: `Type()`** - It handles 90% of cases correctly.

Only use other methods when you have a specific reason:
- `SendKeys() + WaitForText()` - when you know exactly what appears next
- `SendKeys()` alone - only for final actions that don't need waiting
- `SendKeysAndWaitForChange()` - **never** (it's problematic, use `Type()` instead)
