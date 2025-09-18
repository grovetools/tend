# Usage Patterns Section Prompt

You are an expert Go developer and technical writer. Your task is to document common usage patterns for the 'grove-tend' testing library based on the provided source code context.

## Usage Patterns to Include
1. **Basic Test Setup**: How to create and run a simple test
2. **Parameterized Tests**: Running the same test with different inputs
3. **Resource Management**: Using shared resources across tests
4. **Test Organization**: Structuring large test suites
5. **Context Sharing**: Passing data between test steps
6. **Parallel Execution**: Running tests concurrently
7. **Custom Matchers**: Creating reusable assertion helpers

## Task
- For each pattern, describe when and why to use it.
- Provide clear, practical examples that demonstrate the pattern.
- Include both code examples and command-line usage where appropriate.
- Focus on real-world scenarios that testers will encounter.

## Output Format
The output MUST be a single JSON object. The root key must be `usage_patterns`, which should be an array of pattern objects. Each object should have `name`, `description`, and `example` fields.

Example:
```json
{
  "usage_patterns": [
    {
      "name": "Basic Test Setup",
      "description": "The simplest way to get started with grove-tend is to create a single scenario and run it. This pattern is ideal for smoke tests and quick validations.",
      "example": "// main_test.go\npackage main\n\nimport (\n    \"github.com/mattsolo1/grove-tend/harness\"\n)\n\nvar BasicTest = &harness.Scenario{\n    Name: \"Basic Health Check\",\n    Run: func(ctx *harness.Context) error {\n        // Your test logic here\n        return nil\n    },\n}\n\nfunc main() {\n    harness.RunScenario(BasicTest)\n}"
    }
  ]
}
```