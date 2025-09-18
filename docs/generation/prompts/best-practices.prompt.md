# Best Practices Section Prompt

You are an expert Go developer and technical writer. Your task is to document best practices for using the 'grove-tend' testing library based on the provided source code context.

## Best Practices to Document
1. **Scenario Naming**: Conventions for clear, descriptive test names
2. **Error Handling**: Proper error reporting and debugging
3. **Test Isolation**: Ensuring tests don't interfere with each other
4. **Performance**: Writing efficient tests that run quickly
5. **Maintainability**: Structuring tests for long-term maintenance
6. **Documentation**: Commenting and documenting test code
7. **CI/CD Integration**: Running tests in automated pipelines
8. **Debugging**: Techniques for troubleshooting failing tests

## Task
- For each practice, provide clear guidance and rationale.
- Explain not just the "what" but also the "why" behind each practice.
- Include examples of both good and bad practices where helpful.
- Focus on practical advice that improves test quality and developer experience.

## Output Format
The output MUST be a single JSON object. The root key must be `best_practices`, which should be an array of practice objects. Each object should have `title` and `text` fields.

Example:
```json
{
  "best_practices": [
    {
      "title": "Use Descriptive Scenario Names",
      "text": "Scenario names should clearly describe what is being tested and the expected outcome. Good names make test failures immediately understandable.\n\n**Good**: `TestUserLoginWithValidCredentials`\n**Bad**: `Test1` or `LoginTest`\n\nConsider including the test context, action, and expected result in the name."
    }
  ]
}
```