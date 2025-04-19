Using multiple `process-compose` files lets you to customize a `process-compose` application for different environments or different workflows.

### Understanding multiple Compose files

By default, `process-compose` reads two files, a `process-compose.yml` and an optional `process-compose.override.yml` file. By convention, the `process-compose.yml` contains your base configuration. The override file, as its name implies, can contain configuration overrides for existing processes or entirely new processes.

If a process is defined in both files, `process-compose` merges the configurations using the rules described in [Adding and overriding configuration](#adding-and-overriding-configuration).

To use multiple override files, or an override file with a different name, you can use the `-f` option to specify the list of files. `process-compose` merges files in the order they’re specified on the command line. 

When you use multiple configuration files, you must make sure all paths in the files are relative to the base `process-compose` file (the first `process-compose` file specified with `-f`). This is required because override files need not be valid `process-compose` files. Override files can contain small fragments of configuration. Tracking which fragment of a process is relative to which path is difficult and confusing, so to keep paths easier to understand, all paths must be defined relative to the base file.

### Example use case

#### Different environments

A common use case for multiple files is changing a development `process-compose` app for a production-like environment (which may be production, staging or CI). To support these differences, you can split your `process-compose` configuration into a few different files:

Start with a base file that defines the canonical configuration for the processes.

**process-compose.yml**

```yaml
processes:
  web:
    command: "npm start"
    depends_on:
      db:
        condition: process_started
      cache:
        condition: process_started

  db:
    command: "pg_ctl start -l logfile"

  cache:
    command: "systemctl start redis"
```

In this example the development configuration adds debug flags.

**process-compose.override.yml**

```yaml
processes:
  web:
    environment:
      - "DEBUG=true"

  db:
    command: "pg_ctl start -l logfile -d"

```

When you run `process-compose` it reads the overrides automatically.

Now, it would be nice to use this `process-compose` app in a production environment. So, create another override file (which might be stored in a different git repo or managed by a different team).

**process-compose.prod.yml**

```yaml
processes:
  web:
    environment:
      - "PRODUCTION=true"

  cache:
    environment:
      - "TTL=500"
```

To deploy with this production `process-compose` file you can run

```shell
$ process-compose -f process-compose.yml -f process-compose.prod.yml
```

This deploys all three processes using the configuration in `process-compose.yml` and `process-compose.prod.yml` (but not the dev configuration in `process-compose.override.yml`).

### Adding and overriding configuration

`process-compose` copies configurations from the original process over to the local one. If a configuration option is defined in both the original process and the local process, the local value *replaces* or *extends* the original value.

For single-value options like `command`, `working_dir` or `disabled`, the new value replaces the old value.

original process:

```yaml
processes:
  myprocess:
    # ...
    command: python app.py
```

local process:

```yaml
processes:
  myprocess:
    # ...
    command: python otherapp.py
```

result:

```yaml
processes:
  myprocess:
    # ...
    command: python otherapp.py
```

For the **multi-value options** `environment`, `depends_on`, `process-compose` merges entries together with locally-defined values taking precedence:

original process:

```yaml
processes:
  myprocess:
    # ...
    environment:
      - "A=3"
      - "C=8"
```

local process:

```yaml
processes:
  myprocess:
    # ...
    environment:
      - "A=4"
      - "B=5"
```

result:

```yaml
processes:
  myprocess:
    # ...
    environment:
      - "A=4"
      - "B=5"
      - "C=8"
```

### Configuration Inheritance with `extends`

`process-compose` provides the `extends` keyword to simplify configuration file inheritance:

```yaml
# ./some/dir/process-compose.prod.yaml
version: "0.5"
extends: "process-compose.yaml"

processes:
```

```yaml
# ./some/dir/process-compose.yaml
version: "0.5"

processes:
```

This is equivalent to running:

```shell
$ process-compose -f ./some/dir/process-compose.yaml -f ./some/dir/process-compose.prod.yaml
```

And allows you to use the shorter command:

```shell
$ process-compose -f ./some/dir/process-compose.prod.yaml
```

With the same result.

**Notes**:

1. Inheritance chains are limited only by available memory.
2. Circular inheritance will cause loading to fail.
3. The `extends` path is relative to the extending file's location (as shown in the example above).
4. Absolute paths are automatically detected and used as-is.
5. The `.env` file is loaded only from the `CWD`. Additional env files can be specified using `--env` (`-e`).
6. If file `B` uses the `extends` keyword to extend file `A`, loading both with `process-compose up -f A -f B` will fail. Load only the last file in the chain with `process-compose -f B` instead.

## Controlling Process Enabled Status with `is_disabled`

### The Challenge: Overriding `disabled: false`

In `process-compose`, you can define processes in multiple configuration files (e.g., a base `process-compose.yaml` and an `override.yaml`). These files are merged to produce the final configuration.

A common scenario is wanting to disable a process by default in the base configuration and then enable it specifically in an override file. The natural way to attempt this would be:

**`base.yaml`:**

```yaml
processes:
  my_process:
    command: "echo 'running process'"
    disabled: true
```

**`override.yaml`:**

```yaml
processes:
  my_process:
    # Attempt to enable the process
    disabled: false
```

However, due to the underlying configuration merging library (`mergo`) and how Go handles boolean types, this approach has limitations. In Go, the "zero value" for a boolean is `false`. When merging, the library might struggle to distinguish between a `disabled` field that was explicitly set to `false` in the override file and a `disabled` field that was simply *not present* in the override file (which would also result in a `false` value after initialization). This ambiguity can lead to the override `disabled: false` not reliably enabling the process as intended.

### The Solution: Introducing `is_disabled` (string)

To provide a reliable way to enable or disable processes via overriding configuration files, we have introduced a new configuration option: `is_disabled`.

**Key characteristics of `is_disabled`:**

1. **Type:** It is a **string**, not a boolean.
2. **Values:** It accepts the string values `"true"` or `"false"`.
3. **Purpose:** It allows you to explicitly signal your intent to enable or disable a process during configuration merging, overcoming the boolean zero-value ambiguity.

### How to Use `is_disabled`

You can now use `is_disabled` in your override files to reliably control the process state:

**`base.yaml`:**

```yaml
processes:
  my_process:
    command: "echo 'running process'"
    # Can be disabled using either field in the base file
    disabled: true
    # or is_disabled: "true"
```

**`override.yaml` (to enable the process):**

```yaml
processes:
  my_process:
    # Use is_disabled as a string to reliably enable
    is_disabled: "false"
```

**`override.yaml` (to explicitly disable the process, if it wasn't already):**

```yaml
processes:
  my_process:
    # Use is_disabled as a string to reliably disable
    is_disabled: "true"
```

Because `"false"` is not the zero value for a string (which is `""`), the merging logic can clearly distinguish between an explicitly set `is_disabled: "false"` and an unset field.

### Relationship Between `disabled` and `is_disabled`

Both `disabled` (boolean) and `is_disabled` (string) can coexist, but `is_disabled` takes precedence:

1. **If `is_disabled` is set** (to either `"true"` or `"false"` in the final merged configuration): Its value determines whether the process is disabled or enabled. The value of the `disabled` field is ignored.
2. **If `is_disabled` is \*not\* set:** The value of the `disabled` boolean field determines the state. If `disabled` is also not set, the process defaults to being enabled (`disabled: false`).

**Recommendation:**

- For **standard configuration** within a single file or when not dealing with complex override scenarios to *enable* a process, you can continue using the boolean `disabled` field (`disabled: true` or `disabled: false`).
- When **using override files** specifically to *enable* a process that might be disabled in a base file, use `is_disabled: "false"` in the override file for guaranteed results.
- You can also use `is_disabled: "true"` in override files if you prefer consistency, although `disabled: true` generally works reliably for disabling.