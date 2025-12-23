## TUI Interaction

After launching a TUI application into a `Session`, you can interact with it programmatically by sending keystrokes and waiting for the UI to update. The `tend` framework provides high-level APIs to make these interactions reliable and declarative.

### 1. Sending Keystrokes

#### `session.SendKeys`

The `session.SendKeys` method sends a sequence of raw keystrokes to the TUI without waiting for a response. This is a low-level function useful for situations where you don't want to wait for the UI to stabilize after sending input.

Arguments are sent as a single sequence. Special keys like `Enter`, `Esc`, and control combinations (`C-c`) are supported.

```go
// From pkg/tui/session.go
func (s *Session) SendKeys(keys ...string) error {
	// ... sends keys to tmux session
}
```

**Example:**
```go
// Send 'j' twice, then 'Enter'
err := session.SendKeys("j", "j", "Enter")
```

#### `session.Type` (Recommended)

The `session.Type` method is the preferred way to send input. It is a higher-level function that combines sending keystrokes with waiting for the UI to become stable. This single method replaces the common pattern of calling `session.SendKeys(...)` followed by `session.WaitStable()`.

It also includes special handling for Vim-style chord commands. When two single-character keys are passed (e.g., `g`, `g`), it sends them individually with stabilization between each keypress to ensure the TUI correctly interprets the chord.

```go
// From pkg/tui/session.go
func (s *Session) Type(keys ...string) error {
	// ... sends keys and then calls s.WaitStable()
}
```

**Examples:**
```go
// Navigate down one line and wait for the screen to update
if err := session.Type("j"); err != nil {
    return err
}

// Go to the top of a list in a vim-like TUI
if err := session.Type("g", "g"); err != nil {
    return err
}
```

### 2. Waiting for Stability (`session.WaitStable`)

After sending input to a TUI, the interface may perform animations, load data asynchronously, or re-render over several frames. Using a fixed delay like `time.Sleep()` is unreliable and can lead to flaky tests.

The `session.WaitStable` method provides a robust solution. It polls the screen content at a regular interval and waits for it to remain unchanged for a specified duration. This ensures that the UI has finished updating before the test proceeds to the next assertion or interaction.

By default, it uses a 10-second timeout, polls every 100ms, and considers the UI stable after 200ms of no changes.

```go
// From pkg/tui/session.go
func (s *Session) WaitStable() error {
	return s.WaitForUIStable(10*time.Second, 100*time.Millisecond, 200*time.Millisecond)
}
```

**Example:**
```go
// Old, unreliable way
session.SendKeys("Enter")
time.Sleep(500 * time.Millisecond) // May be too short or too long

// New, reliable way
session.SendKeys("Enter")
if err := session.WaitStable(); err != nil {
    return err
}
// The TUI is now guaranteed to be in a steady state
```