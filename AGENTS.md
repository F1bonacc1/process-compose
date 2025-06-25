## Build, Lint, and Test

- Build: `go build -o bin/process-compose ./src`
- Test: `go test -cover ./src/...`
- Test single test: `go test -run ^TestMyTest$ ./src/...`
- Lint: `make lint`

## Code Style

- Imports: Sorted alphabetically.
- Formatting: `gofmt` before committing.
- Types: Use static typing.
- Naming: camelCase for variables, PascalCase for public functions/structs.
- Error Handling: Use `log` for errors, not `fmt.Errorf`.
- Comments: Explain non-obvious code.
- Line Length: Keep lines under 120 characters.
- Concurrency: Use channels for goroutine communication.
- Dependencies: Use Go modules.
- Logging: Use the `slog` library.
- Indentation: Use spaces, not tabs.
- Spacing: Use 4 spaces for Go, 2 for YAML.
- Commits: Follow conventional commit format.
- Pull Requests: Include a detailed description and test plan.
- Documentation: Keep `README.md` and `docs/` updated.
