---
sidebar_position: 8
---

# MCP Server Integration

Process Compose now supports the Model Context Protocol (MCP), allowing you to expose processes as MCP tools and resources. This enables AI assistants and other MCP clients to invoke processes dynamically through a standardized protocol.

## What is MCP?

The Model Context Protocol (MCP) is an open protocol that standardizes how applications provide context to Large Language Models (LLMs). With MCP, you can:

- Expose your process-compose processes as tools that AI assistants can invoke
- Create resource endpoints for configuration files and data
- Integrate with popular AI clients like Cursor, Claude Desktop, and others

## Configuration

### Enabling the MCP Server

Add an `mcp_server` section to your process-compose configuration file:

```yaml
mcp_server:
  host: localhost      # Required: host to bind to (ignored for stdio)
  port: 3000           # Required: port to listen on (ignored for stdio)
  # transport: sse    # Optional: defaults to "sse". Supported: "sse", "stdio"
```

> :bulb: **Both SSE and Stdio transports are supported.** The transport defaults to SSE when not specified, making it optional in your configuration.
>
> The MCP server will only start if at least one process has an `mcp:` configuration section. If you configure `mcp_server` but have no MCP processes, the server will not start.

### Timeout Configuration

You can configure timeouts for MCP tool and resource execution at two levels:

**Global timeout** (applies to all MCP processes):

```yaml
mcp_server:
  host: localhost
  port: 3000
  timeout: "5m"  # Optional: global timeout (default: 5m)
```

**Per-process timeout** (overrides global):

```yaml
processes:
  slow-task:
    command: "sleep 30 && echo done"
    disabled: true
    mcp:
      type: resource
      timeout: "10s"  # This process has 10 second timeout
```

Timeout values use Go duration format (e.g., `"30s"`, `"5m"`, `"1h"`). If a process exceeds its timeout, it will be terminated and an error returned to the MCP client.

## Defining MCP Processes

### MCP Tools

Tools are parameterized processes that accept arguments:

```yaml
processes:
  search-logs:
    command: "grep @{pattern} @{filename}"
    description: "Search for a pattern in a log file"
    disabled: true  # MCP processes must be disabled initially
    working_dir: "/var/log"
    mcp:
      type: tool
      arguments:
        - name: pattern
          type: string
          description: "Search pattern to find in file"
          required: true
        - name: filename
          type: string
          description: "File to search in"
          required: true
```

**Important:** MCP processes must have `disabled: true` since they are invoked on-demand.

#### Argument Types

Supported argument types:

| Type | Description | Example |
| ------ | ------------- | --------- |
| `string` | Text values | `"hello world"` |
| `integer` | Whole numbers | `42` |
| `number` | Floating-point numbers | `3.14` |
| `boolean` | true/false values | `true` |

#### Required and Optional Arguments

Arguments can be marked as required or optional:

```yaml
arguments:
  - name: pattern
    type: string
    required: true     # Must be provided when invoking the tool
  - name: limit
    type: integer
    required: false    # Optional - can be omitted
```

- **Required arguments** (`required: true`): Must be provided when the tool is invoked. If missing, the invocation will fail with an error.
- **Optional arguments** (`required: false` or omitted): Can be omitted. When not provided, the `@{arg}` placeholder is replaced with an empty string.

> :bulb: Optional arguments are substituted with an empty string when not provided. Design your commands to handle this gracefully, or use default values (see below).

#### Argument Substitution

Arguments are substituted directly into the command using the `@{argument_name}` syntax:

```yaml
command: "grep @{pattern} @{filename}"
arguments:
  - name: pattern
    type: string
  - name: filename
    type: string
```

When the tool is invoked with `pattern="error"` and `filename="/var/log/app.log"`, the command becomes:

```bash
grep "error" "/var/log/app.log"
```

**Type-aware formatting:**

- **Strings** are always double-quoted and properly escaped: `"hello world"`, `"say \"hi\""`
- **Integers** and **Numbers** are unquoted: `42`, `3.14`
- **Booleans** are unquoted: `true`, `false`

This approach allows you to mix MCP arguments with regular environment variables:

```yaml
command: "grep @{pattern} $HOME/logs/@{filename}"
```

In this example:

- `@{pattern}` and `@{filename}` are MCP arguments (substituted with actual values)
- `$HOME` is a regular environment variable (handled by the shell)

#### Default Values

You can provide default values for optional arguments using the `@{arg:default}` syntax:

```yaml
processes:
  tail-logs:
    command: "tail -n @{lines:100} @{filename}"
    description: "Show last N lines of a log file"
    disabled: true
    mcp:
      type: tool
      arguments:
        - name: filename
          type: string
          required: true
        - name: lines
          type: integer
          required: false
```

In this example:

- If `lines` is provided (e.g., `50`), the command becomes: `tail -n 50 /path/to/file`
- If `lines` is not provided, the default is used: `tail -n 100 /path/to/file`

You can also define defaults in the argument configuration:

```yaml
arguments:
  - name: timeout
    type: integer
    required: false
    default: "30"  # Default value as a string
```

**Priority order:**

1. Value provided during invocation (highest priority)
2. Default value in `@{arg:default}` pattern
3. Default value from argument definition (`default` field)

#### Escaping

If you need to include literal `@{...}` text in your command (not as an argument placeholder), escape it with a backslash:

```yaml
command: "echo 'Use \@{name} syntax for placeholders'"
```

This will produce: `Use @{name} syntax for placeholders` (without substitution)

### MCP Resources

Resources are parameterless processes that execute on demand:

```yaml
processes:
  get-config:
    command: "cat /etc/myapp/config.json"
    description: "Read application configuration"
    disabled: true
    mcp:
      type: resource
```

Resources are accessed via URIs in the format: `process://<process-name>`

## Complete Examples

### Log Analysis Tool

```yaml
mcp_server:
  host: localhost
  port: 3000

processes:
  analyze-logs:
    command: "tail -n @{lines} @{logfile} | grep @{level}"
    description: "Analyze recent log entries"
    disabled: true
    mcp:
      type: tool
      arguments:
        - name: lines
          type: integer
          description: "Number of lines to analyze"
          required: true
        - name: logfile
          type: string
          description: "Path to log file"
          required: true
        - name: level
          type: string
          description: "Log level to filter (ERROR, WARN, INFO)"
          required: false
```

### System Information Resource

```yaml
processes:
  system-info:
    command: "echo 'CPU: $(nproc), Memory: $(free -h | awk '/^Mem:/ {print $2}'), Disk: $(df -h / | tail -1 | awk '{print $2}')'"
    description: "Get system resource information"
    disabled: true
    mcp:
      type: resource
```

### Database Query Tool

```yaml
processes:
  query-db:
    command: "psql -d mydb -c \"@{query}\""
    description: "Execute a database query"
    disabled: true
    working_dir: "/app"
    mcp:
      type: tool
      arguments:
        - name: query
          type: string
          description: "SQL query to execute"
          required: true
```

## Using with MCP Clients

Process Compose supports both HTTP Server-Sent Events (SSE) and Standard Input/Output (stdio) transports, allowing integration with a wide variety of clients.

### Stdio Clients (Cursor, Claude Desktop, VS Code)

Most popular AI clients and IDEs primarily use stdio transport. To use them with process-compose:

1. Update your configuration to use `transport: stdio`:

```yaml
mcp_server:
  transport: stdio
```

1. Configure your client to run process-compose. For example, in Claude Desktop `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "process-compose": {
      "command": "process-compose",
      "args": ["-f", "/path/to/your/process-compose.yaml"]
    }
  }
}
```

> :bulb: When using `stdio` transport, the TUI is automatically disabled. The TUI and stdio transport cannot be used at the same time as they both require control over the process's standard input and output streams.

### MCP Inspector

Test your MCP configuration with the official MCP Inspector:

**For SSE transport:**

```bash
# Start process-compose with MCP enabled (SSE default)
process-compose -f your-config.yaml

# In another terminal, connect inspector to the SSE endpoint
npx @modelcontextprotocol/inspector http://localhost:3000/sse
```

**For Stdio transport:**

```bash
# Provide the process-compose command directly to the inspector
npx @modelcontextprotocol/inspector process-compose -f your-config.yaml
```

### HTTP/SSE Clients

Any MCP client that supports HTTP/SSE transport can connect to:

```HTTP
http://<host>:<port>/sse
```

For example, with the default configuration:

```HTTP
http://localhost:3000/sse
```

## Behavior and Lifecycle

### Execution Flow

1. **Startup**: MCP server starts alongside the regular process-compose server
2. **Registration**: All MCP-enabled processes are registered as tools/resources
3. **Invocation**: When called via MCP:
   - Arguments are substituted into the command using `@{arg}` syntax
   - Process state changes from "Disabled" to "Pending" to "Running"
   - Process executes with the substituted command and captures output
   - Output is returned to the MCP client
   - Process transitions to "Completed"

### Queued Execution

MCP processes use a queuing mechanism:

- Only one invocation runs at a time per process
- Concurrent requests are queued and executed sequentially
- This prevents conflicts and resource contention

### Process State

After execution, MCP processes remain in the "Completed" state:

- They can be invoked again (process restarts)
- Output is available in the log buffer
- Exit codes are reported back to MCP clients

### Integration with Regular Processes

MCP processes work alongside regular processes:

- They respect dependencies (if a process has `depends_on`, dependencies start first)
- They appear in the TUI and are visible in process lists
- They can be started/stopped manually like any other process
- They participate in ordered shutdown if configured

## Validation

Configuration is validated at startup:

- ✅ MCP server settings (transport, host, port)
- ✅ Process MCP types (tool/resource)
- ✅ Argument types and required fields
- ✅ Tool arguments match `@{arg}` references in commands
- ✅ Resources don't have arguments

Validation errors are logged. In strict mode (`is_strict: true`), invalid configurations will prevent startup.

## Limitations

- **Single-threaded**: Each MCP process handles one invocation at a time
- **No streaming**: Output is returned after process completion
- **Buffer limits**: Output size is limited by the per-process log buffer (`log_length`)

## Troubleshooting

### Process not appearing in MCP client

1. Verify the process has `disabled: true`
2. Ensure the process has a valid `mcp` section
3. Check that `mcp_server.host` and `mcp_server.port` are configured
4. Check logs for validation errors

### Arguments not being passed

1. Verify argument names match the `@{arg}` references in the command
2. Ensure argument names are lowercase in the YAML config (e.g., `pattern`) even if they refer to uppercase-looking variables in examples
3. Check that the `@{arg}` syntax is correct - no spaces inside the braces
4. Ensure arguments are marked as `required: true` if needed

### Process hangs or times out

1. Check if process has proper `depends_on` configuration
2. Verify the command is executable and returns

### Validation errors

Run with `--dry-run` to validate configuration:

```bash
process-compose -f your-config.yaml --dry-run
```

## Best Practices

1. **Always disable MCP processes**: Set `disabled: true` for all MCP processes
2. **Use descriptive names**: Process names become tool/resource names in MCP clients
3. **Add descriptions**: Help users understand what each tool/resource does
4. **Validate arguments**: Handle missing or invalid arguments gracefully in your commands
5. **Set working directories**: Use `working_dir` for processes that need specific contexts
6. **Use absolute paths**: When possible, use absolute paths in commands
7. **Test locally**: Use MCP Inspector to test before connecting to AI clients

## Example Configuration

See the full example in the repository:

```bash
process-compose -f examples/mcp-example.yaml
```

This example demonstrates:

- Multiple tool types with different arguments
- Resource endpoints
- Integration with regular processes
- Best practices for configuration
