# Grove Tend Documentation Generation Prompt

You are an expert Go developer and technical writer specializing in testing frameworks. Your task is to create a comprehensive guide to the 'grove-tend' testing library based on the provided source code examples.

## Output Format

The output MUST be a single XML file with the following structure:

```xml
<tend_guide>
  <introduction>
    A brief, high-level overview of grove-tend, its purpose, and core philosophy. 
    Synthesize this from the provided README.md.
  </introduction>

  <core_concepts>
    <concept name="[Concept Name]">
      <description>Explain what this concept is and its role.</description>
      <example>
        <![CDATA[
        // Provide a minimal, canonical Go snippet
        ]]>
      </example>
    </concept>
  </core_concepts>

  <usage_patterns>
    <pattern name="[Pattern Name]">
      <description>Describe the pattern and when to use it.</description>
      <example>
        <![CDATA[
        // Show a practical example
        ]]>
      </example>
    </pattern>
  </usage_patterns>

  <best_practices>
    <practice title="[Practice Title]">Description of the best practice.</practice>
  </best_practices>
</tend_guide>
```

## Core Concepts to Document

1. **Scenario** - The fundamental unit of a test
2. **Step** - A single action within a Scenario  
3. **Context** - The state container for sharing data between steps

## Usage Patterns to Include

Focus on these key patterns found across Grove ecosystem projects:

1. **Basic File Operations** - Using the `fs` helper package
2. **Command Execution** - Using the `command` helper package
3. **Git Operations** - Using the `git` helper package
4. **Mocking Dependencies** - Using `harness.SetupMocks` and `ctx.Command()`
5. **Swapping Mocks for Real Dependencies** - The `--use-real-deps` flag
6. **Interactive and Debug Modes** - Using `-i` and `-d` flags

## Guidelines

- Analyze ALL provided files, including READMEs, Makefiles, and Go source code
- Identify unique and canonical usage patterns across projects
- Synthesize patterns into clean, minimal examples
- Don't just copy-paste large blocks - distill into perfect, minimal examples
- Include both Go code examples and shell command examples where appropriate
- Focus on real-world usage patterns, not theoretical concepts
- Ensure examples are complete and runnable
- Use descriptive variable names that make examples self-documenting

## Additional Patterns to Look For

When analyzing the test files, pay special attention to:
- How projects structure their test directories
- Common helper functions and utilities
- Error handling patterns
- Assertion patterns using the `assert` package
- How mocks are implemented and used
- Integration with Grove ecosystem tools
- Use of tmux for debugging
- Worktree management patterns
- Docker container management
- Parallel test execution patterns

Remember: The goal is to create documentation that helps developers quickly understand and use grove-tend effectively in their own projects.