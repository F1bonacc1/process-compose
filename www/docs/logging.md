# Logger
## Per Process Log Collection

```yaml
process2:
  log_location: ./pc.process2.log #if undefined or empty no logs will be saved
```

Captures StdOut and StdErr output

## Merge into a single file (Unified Logging)

```yaml
environment:
  - "ABC=42"
log_location: ./pc.global.log #if undefined or empty, no logs will be saved (if not defined per process)
processes:
  process2:
    command: "chmod 666 /path/to/file"
```

## Process compose console log level

```yaml
log_level: info # other options: "trace", "debug", "info", "warn", "error", "fatal", "panic"
processes:
  process2:
    command: "chmod 666 /path/to/file"
```

This setting controls the `process-compose` log level. The processes log level should be defined inside the process. It is recommended to support this definition with an environment variable in `process-compose.yaml`

## Log Rotation

```yaml
# unified log
version: "0.5"
log_level: info 
log_location: /tmp/pc.log
log_configuration:
  rotation:
    max_size_mb: 1  # the max size in MB of the logfile before it's rolled
    max_age_days: 3 # the max age in days to keep a logfile
    max_backups: 3  # the max number of rolled files to keep
    compress: true  # determines if the rotated log files should be compressed using gzip. The default is false

#process level logging (same syntax)
processes:
  someProc:
    command: "some command"
    log_configuration:
      rotation:
        max_size_mb: 1  # the max size in MB of the logfile before it's rolled
        max_age_days: 3 # the max age in days to keep a logfile
        max_backups: 3  # the max number of rolled files to keep
        compress: true  # determines if the rotated log files should be compressed using gzip. The default is false
```

## Logger Configuration

```yaml
log_configuration:
  fields_order: ["time", "level", "message"] # order of logging fields. The default is time, level, message
  disable_json: true                         # output as plain text. The default is false
  timestamp_format: "06-01-02 15:04:05.000"  # timestamp format. The default is RFC3339
  no_metadata: true                          # don't log process name and replica number
  add_timestamp: true                        # add timestamp to the logger. Default is false
  no_color: true	                         # disable ANSII colors in the logger. Default is false
```

| Parameter Name     | Description                                                  | Depends On                                                   | Default Value                                                |
| ------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `fields_order`     | Order of the logging fields. The default is time, level, message. In case one of the fields is omitted, it will be missing in the log as well. | `disable_json: true`<br />`add_timestamp: true` for `"time"` | `["time", "level", "message"]`                               |
| `disable_json`     | Disables JSON logging format. Use *Console Mode Format*.     |                                                              | `false`                                                      |
| `timestamp_format` | Sets the format of the logger [timestamp](https://pkg.go.dev/time#pkg-constants). | `add_timestamp: true`                                        | If `disable_json: true`:`3:04PM`<br />If `disable_json: false`:`"2006-01-02T15:04:05Z07:00"` |
| `no_metadata`      | Don't log the process name and replica number.               |                                                              | `false`                                                      |
| `add_timestamp`    | Add timestamp to the logger. Useful for processes without an internal logger. |                                                              | `false`                                                      |
| `no_color`         | Disable ANSII colors in the log file.                        | `disable_json: true`                                         | `false`                                                      |

## Process Compose Internal Log

Default log location: `/tmp/process-compose-$USER.log`

> :bulb: It is recommended to add the following process configuration to your `process-compose.yaml`:

```yaml
processes:
  pc_log:
    command: "tail -f -n100 process-compose-${USER}.log"
    working_dir: "/tmp"
```

This will allow you to spot any issues with the processes execution, without leaving the `process-compose` TUI.
