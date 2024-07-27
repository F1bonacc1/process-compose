## Set up your machine
Process Compose is written in Go.

### Prerequisites:

- [Make](https://www.gnu.org/software/make/)
- [Go 1.22+](https://go.dev/doc/install)

### Clone Process Compose:

```
git clone git@github.com:F1bonacc1/process-compose.git
```

`cd` into the directory and install the dependencies:

```
go mod tidy
```

You should then be able to build the binary:

```
make build
```

## Test your change

You can create a branch for your changes and try to build from the source as you go.

When you are satisfied with the changes, we suggest you run:

```
make ci
```

## Create a commit

Commit messages should be well formatted, and to make that "standardized", we are using Conventional Commits.

You can follow the documentation on [their website](https://www.conventionalcommits.org/).

## Submit a pull request

Push your branch to your `process-compose` fork and open a pull request against the main branch.

