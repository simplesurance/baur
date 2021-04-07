# baur [![CircleCI](https://circleci.com/gh/simplesurance/baur.svg?style=svg&circle-token=8bc17577e45f5246cba2e1ea199ae504c8700eb6)](https://circleci.com/gh/simplesurance/baur) [![Go Report Card](https://goreportcard.com/badge/github.com/simplesurance/baur)](https://goreportcard.com/report/github.com/simplesurance/baur)

<img src="https://github.com/simplesurance/baur/wiki/media/baur.png" width="256" height="256">

## Content

* [About](#about)
* [Quickstart](#quickstart)
* [Key Features](#key-features)
* [Documentation](#documentation)
* [Example Repository](#example-repository)

## About

baur is an incremental task runner for for git based [monolithic
repositories](https://en.wikipedia.org/wiki/Monorepo).

Per application tasks are defined in a [TOML](https://github.com/toml-lang/toml)
configuration file. Each task specifies:

- a command to run
- the inputs of the task
- optionally artifacts that the task produces (files or docker images) and
  to where tu upload these outputs

When a task is run, baur calculates a digest of it's input and stores it in a
PostgreSQL database.
When baur runs again it only runs tasks for which the inputs have changed.

## Quickstart

This chapter describes a quick way to setup baur for experimenting with it
without using the Example Repository.

### Installation

The recommended version is the latest from the
[release page](https://github.com/simplesurance/baur/releases). \
The master branch is the development branch and might be in an unstable state.

After downloading a release archive, extract the `baur` binary from the archive
(`tar xJf baur-OS_ARCH-VERSION.tar.xz`) and move it to a directory that is
listed in your `$PATH` environment variable.

### Setup

baur uses a PostgreSQL database to record information about past task runs. The
quickest way to setup a PostgreSQL for local testing is with docker:

```sh
docker run -p 5432:5432 -e POSTGRES_DB=baur postgres:latest
```

Afterwards your are ready to create your baur repository configuration.
In the root directory of your Git repository run:

```sh
baur init repo
```

The command will print instructions how to initialize your database and create
your first application configuration file.

### First Steps

Some commands to start with are:

| command                               | action                                                                                               |
|:--------------------------------------|------------------------------------------------------------------------------------------------------|
| `baur status`                         | List applications in the repository with their build status                                          |
| `baur run`                            | Run all tasks of all applications with pending builds, upload their artifacts and records the result |
| `baur ls tasks all`                   | List recorded tasks                                                                                  |
| `baur show currency-service`          | Show information about an application called "currency-service"                                      |
| `baur ls inputs --digests shop.build` | Show inputs with their digests of the build tasks of an application called "shop"                    |

To get more information about a command pass the `--help` parameter to baur.

## Key Features

* **Detecting Changed Applications**
  The inputs of tasks are specified in the `.app.toml` config file for
  each application. baur calculates a SHA384 digest for all inputs and stores
  the digest and information about uploaded artifacts in it's database.
  The digest is used to detect if a previous run of the task for the same input
  files exists. If a run exist, the task does not need to be rerun.
  This allows to only run tasks in CI pipeline for applications that changed in
  a given commit.

* **Artifact Upload to S3 and Docker Registries**
  baur supports uploading file artifacts to S3 buckets and docker images to
  docker registries.

* **Managing Applications**
  baur can be used as management tool in monorepositories to query basic
  information about applications and storage locations of artifacts.

* **CI Optimized:**
  baur is aimed to be run in CI environments and allows to print relevant output
  in CSV format to be easily parsed by scripts.

* **Statistics:**
  The data that baur stores in its PostgreSQL database enables the graphing of
  statistics about task runs (like builds) such as which application changes
  most, which tasks produced the biggest artifacts, which task runs the longest.

## Documentation

Documentation is available in the [wiki](https://github.com/simplesurance/baur/wiki).

## Example Repository

You can find an example monorepository that uses baur at:
<https://github.com/simplesurance/baur-example>.
Please follow the [quickstart guide](https://github.com/simplesurance/baur-example#quickstart)
for the example repository.
