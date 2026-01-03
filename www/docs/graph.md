# Process Dependency Graph

Process Compose provides a powerful way to visualize your process dependencies through multiple interfaces.

## TUI Graph View

You can access an interactive dependency tree directly within the TUI.

- **Shortcut**: `Ctrl+Q`
- **Features**:
  - Interactive expansion/collapse of nodes (use **Enter** or **Mouse Click**).
  - **Status-colored nodes**:
    - <span style="color:green">●</span> **Green**: Running
    - <span style="color:yellow">●</span> **Yellow**: Pending, Launching
    - <span style="color:blue">●</span> **Blue**: Completed
    - <span style="color:red">●</span> **Red**: Error, Failed
    - <span style="color:white">●</span> **White**: Other statuses
  - Indication of dependency conditions (e.g., `<process_healthy>`).

## CLI Graph Command

The `graph` command allows you to export or view the dependency tree in various formats.

```shell
# Display a beautiful ASCII tree (default)
process-compose graph

# Export to Mermaid flowchart format
process-compose graph --format mermaid

# Get the raw graph data in JSON or YAML
process-compose graph --format json
process-compose graph --format yaml
```

### Examples

#### ASCII Output
```text
Dependency Graph
└── frontend [Pending]
    └── api-server [Running] <process_healthy>
        ├── postgres [Running] <process_healthy>
        └── redis [Running] <process_healthy>
```

#### Mermaid Integration
You can pipe the mermaid output directly to a file or a viewer:
```shell
process-compose graph -f mermaid > graph.mmd
```

## REST API

The dependency graph is also available via the REST API, providing a recursive JSON structure of all processes and their dependencies.

- **Endpoint**: `GET /graph`
- **Response**: A JSON object containing a `nodes` map where each entry represents a "leaf" (process that no other process depends on) and its full recursive dependency tree.

```bash
curl localhost:8080/graph
```
