## Process Compose

[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](https://go.dev/) [![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://GitHub.com/F1bonacc1/process-compose/graphs/commit-activity) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com) ![Go Report](https://goreportcard.com/badge/github.com/F1bonacc1/process-compose) [![Releases](https://img.shields.io/github/downloads/F1bonacc1/process-compose/total.svg)]() ![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/ProcessCompose?style=flat-square&logo=twitter&logoColor=white)


Process Compose is a simple and flexible scheduler and orchestrator to manage non-containerized applications.

**Why?** Because sometimes you just don't want to deal with docker files, volume definitions, networks and docker registries.

#### Features:

- Processes execution (in parallel or/and serially)
- Processes dependencies and startup order
- Process recovery policies
- Manual process [re]start
- Processes arguments `bash` or `zsh` style (or define your own shell)
- Per process and global environment variables
- Per process or global (single file) logs
- Health checks (liveness and readiness)
- Terminal User Interface (TUI) or CLI modes
- Forking (services or daemons) processes
- REST API (OpenAPI a.k.a Swagger)
- Logs caching
- Functions as both server and client
- Configurable shortcuts
- Merge Configuration Files
- Namespaces
- Run Multiple Replicas of a Process

It is heavily inspired by [docker-compose](https://github.com/docker/compose), but without the need for containers. The configuration syntax tries to follow the docker-compose specifications, with a few minor additions and lots of subtractions.

<img src="./imgs/tui.png" alt="TUI" style="zoom:67%;" />

## Get Process Compose

[Installation Instructions](https://f1bonacc1.github.io/installation/)

## Documentation

[Quick Start](https://f1bonacc1.github.io/intro/)

[Documentation](https://f1bonacc1.github.io/launcher/)

## How to Contribute

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Add some feature')
4. Push to the branch (git push origin my-new-feature)
5. Create new Pull Request

English is not my native language, so PRs correcting grammar or spelling are welcome and appreciated.

## Consider supporting the project ❤️

##### Github (preferred)

https://github.com/sponsors/F1bonacc1

##### Bitcoin

<img src="./imgs/btc.wallet.qr.png" style="zoom:80%;"  alt="3QjRfBzwQASQfypATTwa6gxwUB65CX1jfX"/>
3QjRfBzwQASQfypATTwa6gxwUB65CX1jfX

Thank **You**!
