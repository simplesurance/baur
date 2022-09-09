# baur [![Go Report Card](https://goreportcard.com/badge/github.com/simplesurance/baur)](https://goreportcard.com/report/github.com/simplesurance/baur)

<img src="https://github.com/simplesurance/baur/wiki/media/baur.png" width="256" height="256">

## About

baur is an incremental task runner for [monolithic
repositories](https://en.wikipedia.org/wiki/Monorepo). \
It can be used in CI environments to build, check and test only applications
that changed in a commit.

Practical usage examples of baur can be found in the [example
repository](https://github.com/simplesurance/baur-example).

<a href="https://asciinema.org/a/410274?rows=45" target="_blank"><img src="https://asciinema.org/a/410274.svg" height="250"/></a>

### How it works

Per application tasks are defined in a [TOML](https://github.com/toml-lang/toml)
configuration file. Each task specifies:

- a command to run,
- which inputs (files) affect the result of the task run,
- and optionally artifacts that are created by the task and their upload
  destinations.

When baur runs a task, it calculates a digest for the task inputs and stores it
in a PostgreSQL database. \
On following runs, baur only runs tasks for which the inputs changed.

### Key Features

* **Running Tasks only for Changed Applications** \
  Tasks define which inputs affect the result of the task execution. \
  baur can only runs tasks that have not been run before for the current set of
  inputs.
  \
  Inputs can be defined as 
  [glob file patterns](https://en.wikipedia.org/wiki/Glob_(programming)),
  as strings on the commandline, or as Go package queries.

* **Artifact Upload** \
  Artifacts can be uploaded to **S3** buckets, to **Docker** registries or
  simply copied to another directory in the **filesystem**.

* **Application Management** \
  baur can be used as management tool in monorepositories to query basic
  information about applications and upload destinations for specific builds.

* **CI Optimized** \
  baur is made to be run in CI environments and supports to output information
  in the easily-parseable CSV format.

* **Configuration File Includes** \
  Tasks, Inputs and Output definitions that are shared between tasks can be
  defined in include configuration files.

* **Templating** \
  [Templating](https://github.com/simplesurance/baur/wiki/v2-Configuration#templating-in-configuration-files)
  can be used in configuration files.


## Quickstart

### Installation

#### From a Release

The recommended way is to download the latest released version from the [release
page](https://github.com/simplesurance/baur/releases). \
Official releases are provided for Linux, macOS and Windows.

After downloading the release archive, extract the `baur` binary
(`tar xJf baur-OS_ARCH-VERSION.tar.xz`) and move it to your preferred location.

#### From Source

With Go >= 1.16.0 installed you can build and install the latest version from
the master branch by running:

```sh
go install github.com/simplesurance/baur/v2/...@master
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

## Upgrading from baur 0.20 to 2.0

See [Upgrade Instructions in the wiki](https://github.com/simplesurance/baur/wiki/v2-Upgrade)

## Documentation

Documentation is available in the [wiki](https://github.com/simplesurance/baur/wiki).

## Contributing

We are happy to receive Pull Requests for baur. \
If you like to contribute a non-trivial change, it is recommended to outline the
idea before in the [Ideas forum](https://github.com/simplesurance/baur/discussions/categories/ideas).

## Contact

- Questions? -  [Q&A Forum](https://github.com/simplesurance/baur/discussions/categories/q-a)
- Suggestion for a cool feature or other improvements? - [Ideas Forum](https://github.com/simplesurance/baur/discussions/categories/ideas)

## Links

- [Example Repository](https://github.com/simplesurance/baur-example)
- [Wiki](https://github.com/simplesurance/baur/wiki)
