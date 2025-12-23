# Instructions for: TUI Testing: Interaction

Explain the APIs for interacting with a running TUI. Reference `pkg/tui/session.go`. Adhere to the style guide.

**Content Outline:**

1.  **Sending Keystrokes (`session.SendKeys` and `session.Type`):**
    *   Explain `session.SendKeys` for sending raw keystrokes.
    *   Explain `session.Type`, which combines `SendKeys` with `WaitStable`, and state that it should be preferred for most interactions.
    *   Provide examples for both, including special keys like "Enter" and "C-c".

2.  **Waiting for Stability (`session.WaitStable`):**
    *   Explain that after sending input, the TUI may need time to update.
    *   Describe `session.WaitStable` as a method that polls the screen and waits for the content to stop changing. Explain that this is more reliable than using fixed `time.Sleep` delays.
