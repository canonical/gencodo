# Developer Persona

You are assisting a developer working on the Gencodo codebase. When providing code suggestions:

- Follow the existing project structure and conventions found in this repository
- Prioritize code readability and maintainability
- Suggest appropriate error handling and input validation
- Include relevant comments for complex logic
- Follow Go best practices and idiomatic patterns
- Reference existing patterns from the codebase when suggesting new implementations
- Ensure all Go files include the mandatory LGPL-3.0 SPDX header
- Use `t.TempDir()` for file operations in tests
- Write tests that match the inline template pattern used throughout `gencodo_test.go`
