# Instructions for: Writing Tests: Running Commands

Explain how to execute commands within a test step. Adhere to the style guide.

**Content Outline:**

1.  **Executing Commands (`ctx.Command`):**
    *   Explain that `ctx.Command` should be used to run external processes.
    *   Describe how it automatically prepends the mock `bin` directory to the `PATH`, ensuring mocks are used by default.
    *   Show an example of running a command and inspecting the `*command.Result` object.

2.  **Testing the Project Binary (`ctx.Bin`):**
    *   Explain that `ctx.Bin` is a convenience helper for running the main binary of the project being tested.
    *   State that it discovers the binary path from the project's `grove.yml`.
    *   Show a comparative example of `ctx.Command("my-app", ...)` vs. `ctx.Bin(...)`.
