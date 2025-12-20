# TUI Testing API Evolution

## Problem

The TUI testing API had too many similar methods:

### Before: 7 methods to choose from
- `SendKeys()` - send without waiting
- `SendKeysAndWaitForChange()` - send and wait for change
- `WaitForText()` - wait for specific text
- `WaitForAnyText()` - wait for multiple options
- `WaitForTextPattern()` - wait for regex
- `WaitForUIStable()` - complex wait with parameters
- `WaitStable()` - simple wait wrapper

**Problem**: Users had to understand when to use each method and often made wrong choices.

## Solution: Add `Type()` as the Default

### New Simplified API

```go
// Type() = SendKeys() + WaitStable()
// This is what you want 90% of the time
session.Type("j")  // Navigate and wait

// Old verbose way (still works)
session.SendKeys("j")
session.WaitStable()
```

### Design Principles

1. **Make the common case easy**: `Type()` handles 90% of scenarios
2. **Keep power-user options**: Advanced methods still available when needed
3. **Self-documenting**: Method name clearly indicates behavior
4. **Zero breaking changes**: All existing code continues to work

## Usage Analysis

From analyzing real tests in `scenarios_runner_tui.go`:

- **40+ instances** of `SendKeys() + WaitStable()` ← Perfect for `Type()`
- **10+ instances** of `SendKeys() + WaitForText()` ← Keep for specific cases
- **0 instances** of just `SendKeys()` ← Almost never used alone
- **0 instances** of `SendKeysAndWaitForChange()` ← Removed (problematic)

## When to Use Each Method

### Use `Type()` (default choice)
```go
session.Type("j")           // Navigation
session.Type("g", "g")      // Multi-key commands
session.Type("A")           // Toggles
session.Type("/")           // Open dialogs
```

### Use `SendKeys() + WaitForText()` (specific expectations)
```go
session.SendKeys("/")
session.WaitForText("Search:", 2*time.Second)
```

### Use `SendKeys()` alone (rare - final actions)
```go
session.SendKeys("q")  // Quit - no need to wait
```

### Never use `SendKeysAndWaitForChange()`
It's problematic because it times out if screen doesn't change (which happens with vim commands, navigation at boundaries, etc.).

## Benefits

### Before (verbose and error-prone)
```go
if err := session.SendKeys("j"); err != nil {
    return err
}
if err := session.WaitStable(); err != nil {
    return err
}

// Easy to forget the wait:
session.SendKeys("j")  // Bug! Next line runs before UI updates
session.AssertContains("item-2")
```

### After (concise and correct)
```go
if err := session.Type("j"); err != nil {
    return err
}

// Can't forget to wait - it's built in:
session.Type("j")
session.AssertContains("item-2")  // Always works correctly
```

## Future Improvements

If this proves successful, consider:

1. **Fluent API** (v2.0)
   ```go
   session.Do("j").Wait()
   session.Do("/").WaitFor("Search:")
   ```

2. **Deprecation path**
   - Mark `SendKeysAndWaitForChange()` as deprecated
   - Add linter hints suggesting `Type()` when seeing `SendKeys() + WaitStable()`

3. **Context-aware timeouts**
   ```go
   session.Type("A").WithLongerTimeout()  // For tree rebuilds
   ```

## Migration Guide

### No migration needed!

All existing code continues to work. `Type()` is purely additive.

### Optional migration

Replace this pattern:
```go
session.SendKeys(keys...)
session.WaitStable()
```

With:
```go
session.Type(keys...)
```

Benefits: **50% less code**, **clearer intent**, **impossible to forget waiting**.

## Metrics

- **Code reduction**: 50% (2 lines → 1 line)
- **Methods to learn**: 7 → effectively 2 (`Type()` + special cases)
- **Chance of timing bugs**: Reduced by making correct pattern the easy pattern
- **Breaking changes**: 0
- **Test coverage**: New `TestSession_Type` added

## Documentation

- Quick reference: `docs/tui-testing-quick-reference.md`
- Full proposal: `docs/tui-api-simplification-proposal.md`
- This doc: `docs/tui-api-evolution.md`
