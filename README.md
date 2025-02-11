# baur [![Go Report Card](https://goreportcard.com/badge/github.com/simplesurance/baur)](https://goreportcard.com/report/github.com/simplesurance/baur)

<img src="https://github.com/simplesurance/baur/wiki/media/baur.png" width="256" height="256">

## About

baur is an incremental task runner for [monolithic Git
repositories](https://en.wikipedia.org/wiki/Monorepo). \
It can only run tasks if no previous execution with identical
user-defined inputs (e.g. source files) exists.

It can be used in CI environments to build, check and test only applications
that are affected by changes.

Task outputs like build artifacts can be stored locally and remote.\
Information about task executions are stored in PostgreSQL and can be queried. 

Practical usage examples of baur can be found in the [example
repository](https://github.com/simplesurance/baur-example).

<a href="https://asciinema.org/a/410274?rows=45" target="_blank"><img src="https://asciinema.org/a/410274.svg" height="250"/></a>

### How it works

Task are defined per application in [TOML](https://github.com/toml-lang/toml)
configuration files.\
Definitions can be shared between applications by defining them in include
files. \
Each task specifies:

- A command to execute,
- Inputs that determine if a task needs to be run:
  - Files
  - Environment variables
  - Golang source files, (imported packages are automatically recursively
    resolved to files)
  - Results from other task runs
- Optionally outputs and their upload destinations: 
  - Files (upload to S3 or copy in local filesystem),
  - Docker Images

baur calculates a digest of all task inputs and stores it for successful runs in
the database.
On following runs, baur only runs tasks for which no run with the same digest
exist.

Information about successful task runs are stored in the database and can be
queried, including the URIs of uploaded outputs. \
A set of references of successful task runs can be stored together with
user-defined data (e.g. a changelog) in the database and queried.

## Quickstart

### Installation

#### From a Release

The recommended way is to download the latest released version from the [release
page](https://github.com/simplesurance/baur/releases). \
Official releases are provided for Linux, macOS and Windows.

After downloading the release archive, extract the `baur` binary
(`tar xJf baur-OS_ARCH-VERSION.tar.xz`) and move it to your preferred location.

#### From Source

You can build and install the latest version from the main branch by running:

```sh
go install github.com/simplesurance/baur/v5/...@main
```

### Setup

baur uses a PostgreSQL database to record information about past task runs. The
quickest way to setup a PostgreSQL for local testing is with docker:

```sh
docker run -p 127.0.0.1:5432:5432 -e POSTGRES_DB=baur -e POSTGRES_HOST_AUTH_METHOD=trust postgres:latest
```

Afterwards you create your baur repository configuration file.
In the root directory of your Git repository run:

```sh
baur init repo
```

The command will print instructions how to initialize your database and create
your first application configuration file.

### First Steps

To show information about the available commands run:

```sh
baur --help
```

Some commands to start with are:

| command                               | action                                                                                               |
|:--------------------------------------|------------------------------------------------------------------------------------------------------|
| `baur status`                         | List task in the repository with their build status                                                  |
| `baur run`                            | Run all tasks of all applications with pending builds, upload their artifacts and records the result |
| `baur ls runs all`                    | List recorded tasks                                                                                  |
| `baur show currency-service`          | Show information about an application called *currency-service*                                      |
| `baur ls inputs --digests shop.build` | List inputs with their digests of the *build* task of an application called *shop*                   |
| `baur run --help`                     | Show the usage information for the *run* command.                                                    |

## Documentation

Documentation is available in the [wiki](https://github.com/simplesurance/baur/wiki).

## Upgrading from older baur Versions

See [Upgrade Instructions in the wiki](https://github.com/simplesurance/baur/wiki#upgrade-guide)

## Versioning

baur follows [Semantic Versioning](https://semver.org/) for its command line
interface, configuration file format and database schema. \
The APIs of the Go packages are **excluded** from the semantic versioning policy.
Their APIs may change at any time in a backward incompatible manner.

## Contributing

We are happy to receive Pull Requests for baur. \
If you like to contribute a non-trivial change, it is recommended to outline the
idea before in the [Ideas forum](https://github.com/simplesurance/baur/discussions/categories/ideas).

## Contact

* Questions? - [Q&A Forum](https://github.com/simplesurance/baur/discussions/categories/q-a)
* Suggestion for a cool feature or other improvements? - [Ideas Forum](https://github.com/simplesurance/baur/discussions/categories/ideas)

## Links

* [Example Repository](https://github.com/simplesurance/baur-example)
* [Wiki](https://github.com/simplesurance/baur/wiki)
