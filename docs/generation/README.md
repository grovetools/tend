# Documentation Generation Files

This directory contains the configuration files for generating grove-tend documentation.

## Files

- **`examples.cx.rules`** - Defines which files to include in the context for documentation generation. This includes grove-tend's own examples and test files from across the Grove ecosystem.

- **`tend-docs.prompt.md`** - The customizable prompt that guides the LLM in generating structured documentation. Edit this file to influence what patterns and concepts the documentation focuses on.

## Usage

Run `tend docs generate` from the grove-tend source directory to regenerate the documentation. This will:

1. Build context using `examples.cx.rules`
2. Send the context and prompt to an LLM
3. Generate `pkg/docs/tend-examples.xml` (embedded in the binary)
4. Generate `docs/TEND_GUIDE.md` (human-readable version)

## Customization

- Edit `examples.cx.rules` to include/exclude files from the generation context
- Edit `tend-docs.prompt.md` to guide the LLM's focus and output structure
- The LLM model can be configured in grove.yml under `flow.oneshot_model`

## Best Practices

- Exclude generated files from the input context to avoid feedback loops
- Keep the prompt focused on extracting real patterns from actual code
- Commit both the XML and Markdown files after generation