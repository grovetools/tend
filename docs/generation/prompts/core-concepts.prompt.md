# Core Concepts Section Prompt

You are an expert Go developer and technical writer. Your task is to document the core concepts of the 'grove-tend' testing library based on the provided source code context.

## Core Concepts to Document
1. **Scenario**: The fundamental unit of a test.
2. **Step**: A single action within a Scenario.
3. **Context**: The state container for sharing data between steps.
4. **Harness**: The test runner that orchestrates scenarios.
5. **Resources**: Managed components that scenarios can use.
6. **Tags**: Mechanism for selectively running tests.

## Task
- For each concept, provide a clear description of its role and purpose.
- Explain how the concept fits into the larger testing framework.
- Provide a minimal, canonical Go code snippet as an example for each.
- Focus on practical usage and best practices.

## Output Format
The output MUST be a single JSON object. The root key must be `core_concepts`, which should be an array of concept objects. Each object should have `name`, `description`, and `example` fields.

Example:
```json
{
  "core_concepts": [
    {
      "name": "Scenario",
      "description": "A Scenario is the fundamental unit of a test in grove-tend. It represents a complete test case with setup, execution, and teardown phases.",
      "example": "// Define a basic scenario\nvar MyScenario = &harness.Scenario{\n    Name: \"Test User Login\",\n    Tags: []string{\"auth\", \"smoke\"},\n    Run: func(ctx *harness.Context) error {\n        // Test implementation\n        return nil\n    },\n}"
    }
  ]
}
```