# Introduction Section Prompt

You are an expert Go developer and technical writer. Your task is to write the introduction for the 'grove-tend' testing library based on the provided source code context.

## Task
- Write a comprehensive, high-level overview of grove-tend, its purpose, and core philosophy.
- Synthesize this from the provided README.md and other context files.
- The introduction should be engaging and explain why grove-tend exists and what problems it solves.
- Focus on the unique value proposition and key differentiators.

## Output Format
The output MUST be a single JSON object containing only the introduction text. Do not include any other explanatory text or markdown formatting outside the JSON.

The introduction text itself should be written in Markdown format within the JSON string.

Example:
```json
{
  "introduction": "# Introduction\n\nGrove-tend is a Go library for creating powerful, scenario-based end-to-end testing frameworks..."
}
```