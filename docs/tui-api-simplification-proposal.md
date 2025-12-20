# TUI Testing API Simplification Proposal

## Current Problems

The current API has too many similar methods that confuse users:

### Sending Keys (2 methods)
- `SendKeys(keys...)` - send keys, no waiting
- `SendKeysAndWaitForChange(timeout, keys...)` - send keys and wait for screen to change

### Waiting (5 methods)
- `WaitForText(text, timeout)` - wait for specific text
- `WaitForAnyText(texts, timeout)` - wait for any of several texts
- `WaitForTextPattern(pattern, timeout)` - wait for regex pattern
- `WaitForUIStable(timeout, poll, stable)` - wait for screen to stabilize (complex)
- `WaitStable()` - wait for screen to stabilize (simple wrapper)

**Total: 7 methods** that users need to understand and choose between.

## Observed Usage Patterns

From analyzing actual test code:

```go
// Pattern 1: Send + WaitStable (90% of cases)
session.SendKeys("j")
session.WaitStable()

// Pattern 2: Send + WaitForText (when we know what appears)
session.SendKeys("/")
session.WaitForText("Search:", 3*time.Second)

// Pattern 3: Just send (rare, usually wrong)
session.SendKeys("q")

// Pattern 4: Send + WaitForChange (problematic, should rarely be used)
session.SendKeysAndWaitForChange(2*time.Second, "Enter")
```

## Proposed Simplified API

### Core Principle: Explicit Intent

Instead of many methods, provide **one method with clear options**:

```go
// Single unified method
func (s *Session) Do(keys ...string) *Action

type Action struct {
    session *Session
    keys    []string
    timeout time.Duration
}

// Chainable wait methods
func (a *Action) Wait() error                           // Wait for stability (default)
func (a *Action) WaitFor(text string) error            // Wait for specific text
func (a *Action) WaitForAny(texts ...string) error     // Wait for any text
func (a *Action) WaitForMatch(pattern *regexp.Regexp)  // Wait for pattern
func (a *Action) NoWait() error                         // Don't wait (explicit)
func (a *Action) WithTimeout(d time.Duration) *Action  // Set timeout
```

### Usage Examples

```go
// Most common: send and wait for stability
session.Do("j").Wait()  // Replaces: SendKeys("j") + WaitStable()

// Wait for specific text
session.Do("/").WaitFor("Search:")  // Replaces: SendKeys("/") + WaitForText("Search:", ...)

// Multiple keys
session.Do("g", "g").Wait()  // Replaces: SendKeys("g", "g") + WaitStable()

// Custom timeout
session.Do("j").WithTimeout(5*time.Second).Wait()

// Explicit no-wait (rare)
session.Do("q").NoWait()

// Wait for any of several texts
session.Do("?").WaitForAny("Help", "Usage")
```

### Benefits

1. **Clearer intent**: `Do().Wait()` makes it obvious you're sending keys AND waiting
2. **Fewer decisions**: One entry point (`Do()`) instead of choosing between `SendKeys` variants
3. **Chainable**: Natural fluent API that reads like English
4. **Sensible defaults**: `Wait()` does the right thing 90% of the time
5. **Self-documenting**: `NoWait()` explicitly shows when you're NOT waiting

## Migration Path

### Phase 1: Add new API alongside old

```go
// Old (still works)
session.SendKeys("j")
session.WaitStable()

// New (preferred)
session.Do("j").Wait()
```

### Phase 2: Deprecate old methods

```go
// SendKeys deprecated in favor of Do().NoWait() or Do().Wait()
// @deprecated Use Do(keys...).Wait() instead
func (s *Session) SendKeys(keys ...string) error

// SendKeysAndWaitForChange deprecated - problematic API
// @deprecated Use Do(keys...).Wait() or Do(keys...).WaitFor(text)
func (s *Session) SendKeysAndWaitForChange(timeout, keys...) error
```

### Phase 3: Remove deprecated methods (next major version)

## Alternative: Simpler Improvement (Less Breaking)

If full redesign is too much, just add **one helper** to cover the common case:

```go
// New helper for the 90% case
func (s *Session) Type(keys ...string) error {
    if err := s.SendKeys(keys...); err != nil {
        return err
    }
    return s.WaitStable()
}
```

Then tests become:

```go
// Instead of:
session.SendKeys("j")
session.WaitStable()

// Just:
session.Type("j")
```

This gives 80% of the benefit with minimal changes.

## Recommendation

**Start with the simpler improvement** (add `Type()` helper), then consider the full redesign for v2.0.

The key insight: **90% of the time, you want to send keys AND wait for stability**. Make that the easy path.

## Current Test Pattern Analysis

Looking at `scenarios_runner_tui.go`:
- 40+ instances of `SendKeys() + WaitStable()`
- 10+ instances of `SendKeys() + WaitForText()`
- 0 instances of just `SendKeys()` without waiting
- 0 instances of `SendKeysAndWaitForChange()` (we removed them all!)

This proves: **Always waiting is the right default**.
