---
sidebar_position: 7
---

# Scheduled Processes

Process Compose supports optional scheduling for processes, allowing you to run commands on a cron schedule or at fixed intervals. This is useful for periodic tasks like backups, cleanup jobs, or health checks.

## Configuration

Add a `schedule` section to your process configuration to enable scheduling:

### Cron-Based Scheduling

Use standard 5-field cron expressions to schedule processes:

```yaml
processes:
  daily-backup:
    command: "./backup.sh"
    schedule:
      cron: "0 2 * * *"  # Run daily at 2 AM
      timezone: "UTC"    # Optional: defaults to local timezone
```

### Interval-Based Scheduling

Use Go duration syntax for periodic execution:

```yaml
processes:
  periodic-cleanup:
    command: "./cleanup.sh"
    schedule:
      interval: "30m"      # Run every 30 minutes
      run_on_start: true   # Also run immediately on startup
```

### Schedule Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `cron` | string | - | Cron expression (e.g., `"0 2 * * *"`) |
| `interval` | string | - | Go duration (e.g., `"30m"`, `"1h"`, `"5s"`) |
| `timezone` | string | Local | Timezone for cron (e.g., `"UTC"`, `"America/New_York"`) |
| `run_on_start` | bool | false | Run immediately when process-compose starts |
| `max_concurrent` | int | 1 | Maximum concurrent executions |

> [!NOTE]
> Only one of `cron` or `interval` should be specified. If both are present, `cron` takes precedence.

## Cron Expression Format

The cron expression uses the standard 5-field format:

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, Sunday = 0)
│ │ │ │ │
* * * * *
```

### Examples

| Expression | Description |
|------------|-------------|
| `0 2 * * *` | Every day at 2:00 AM |
| `*/15 * * * *` | Every 15 minutes |
| `0 0 * * 0` | Every Sunday at midnight |
| `0 9-17 * * 1-5` | Every hour from 9 AM to 5 PM, Monday to Friday |
| `0 0 1 * *` | First day of every month at midnight |

## Interval Format

Intervals use Go's duration syntax:

| Suffix | Meaning |
|--------|---------|
| `s` | seconds |
| `m` | minutes |
| `h` | hours |

### Examples

| Interval | Description |
|----------|-------------|
| `5s` | Every 5 seconds |
| `30m` | Every 30 minutes |
| `1h` | Every hour |
| `1h30m` | Every 1 hour and 30 minutes |

## Behavior

### Startup Behavior

By default, scheduled processes do **not** run on startup — they wait for their first scheduled time. To run immediately on startup, set `run_on_start: true`:

```yaml
processes:
  health-check:
    command: "./check-health.sh"
    schedule:
      interval: "5m"
      run_on_start: true  # Run immediately, then every 5 minutes
```

### Overlap Prevention

By default, `max_concurrent: 1` prevents overlapping executions. If a scheduled run triggers while a previous run is still active, the new run is skipped:

```yaml
processes:
  slow-backup:
    command: "./backup.sh"  # Takes 45 minutes
    schedule:
      interval: "30m"
      max_concurrent: 1     # Skip if previous run still active
```

To allow multiple concurrent runs:

```yaml
processes:
  fast-task:
    command: "./quick-task.sh"
    schedule:
      interval: "1m"
      max_concurrent: 3     # Allow up to 3 concurrent executions
```

### Scaling Restrictions

Scheduled processes are designed to be singleton tasks triggered by time. Therefore, **scaling is not supported** for scheduled processes.
- **Configuration**: Setting `replicas` greater than 1 for a scheduled process will result in a validation error (or a warning in non-strict mode).
- **Runtime**: The "Scale" dialog in the TUI is disabled for scheduled processes.

### Manual Schedule Control (Enabling/Disabling)

You can use the `disabled: true` configuration to prevent a scheduled process from starting automatically.

- **Enabling**: A process is "enabled" when it is manually started (via TUI `F7`, CLI, or REST API). This automatically resumes its schedule.
- **Disabling**: A process's schedule is "disabled" (paused) when it is manually stopped. This prevents future scheduled runs until it is manually started again.

```yaml
processes:
  manual-backup:
    command: "./backup.sh"
    disabled: true
    schedule:
      cron: "0 2 * * *"  # Only starts running after first manual start
```

This behavior allows you to "turn off" a scheduled task on demand by stopping it, and "turn it on" by starting it.

### Viewing Next Run Time

The next scheduled run time is displayed in the **Process Info** dialog (press `F3` in the TUI). For scheduled processes, you'll see a "Next Run:" field showing when the process will run next.

The next run time is also available via the REST API in the `next_run_time` field of the process state.

## Complete Examples

### Database Backup

```yaml
processes:
  db-backup:
    command: "pg_dump -U postgres mydb > /backups/mydb-$(date +%Y%m%d).sql"
    schedule:
      cron: "0 3 * * *"    # Daily at 3 AM
      timezone: "UTC"
    working_dir: "/app"
```

### Log Rotation

```yaml
processes:
  log-rotate:
    command: "logrotate /etc/logrotate.conf"
    schedule:
      cron: "0 0 * * *"    # Daily at midnight
```

### Health Check

```yaml
processes:
  health-check:
    command: "curl -f http://localhost:8080/health || exit 1"
    schedule:
      interval: "30s"
      run_on_start: true
```

### Cleanup Job

```yaml
processes:
  temp-cleanup:
    command: "find /tmp -type f -mtime +7 -delete"
    schedule:
      interval: "1h"
      run_on_start: false  # Don't run on startup
```
