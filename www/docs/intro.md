# Getting Started

## Quick Start

Imaginary system diagram:

![Diagram](https://github.com/F1bonacc1/process-compose/raw/main/imgs/diagram.png)

`process-compose.yaml` definitions for the system above:

```yaml
version: "0.5"

environment:
  - "GLOBAL_ENV_VAR=1"
log_location: /path/to/combined/output/logfile.log
log_level: debug

processes:
  Manager:
    command: "/path/to/manager"
    availability:
      restart: "always"
    depends_on:
      ClientA:
        condition: process_started
      ClientB:
        condition: process_started

  ClientA:
    command: "/path/to/ClientA"
    availability:
      restart: "always"
    depends_on:
      Server_1A:
        condition: process_started
      Server_2A:
        condition: process_started
    environment:
      - "LOCAL_ENV_VAR=1"

  ClientB:
    command: "/path/to/ClientB -some -arg"
    availability:
      restart: "always"
    depends_on:
      Server_1B:
        condition: process_started
      Server_2B:
        condition: process_started
    environment:
      - "LOCAL_ENV_VAR=2"

  Server_1A:
    command: "/path/to/Server_1A"
    availability:
      restart: "always"

  Server_2A:
    command: "/path/to/Server_2A"
    availability:
      restart: "always"

  Server_1B:
    command: "/path/to/Server_1B"
    availability:
      restart: "always"

  Server_2B:
    command: "/path/to/Server_2B"
    availability:
      restart: "always"
```

Finally, run `process-compose` in the `process-compose.yaml` directory. Or give it a direct path:

```shell
process-compose -f /path/to/process-compose.yaml
```