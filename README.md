# baur
baur manages builds and artifacts in mono repositories.

When Git repositories contain only a single applications the CI/CD jobs can be
triggered on every new commit.
Monorepositories can contain hundreds of apps and it becomes inefficient and
slow to building, test and deploy all applications on every commit.
A solution is required to detect which applications have changed and run CI/CD
tasks only for those.

baur will implement:
- discovery of applications in a repository,
- management to store and retrieve build artifacts for applications,
- detect if build artifacts for an application version already exist or if it's
  need to be build

baur makes certain Assumptions:
- the baur repository is part of a git repository,
- an application directory only contains one application,
- an application can be build by running a single command,
  a build has to produce 1 or more build artifacts


## Build
To build baur run `make`.

## Dependencies
- The git command lines tools are used to retrieve information in a baur
  repository. The tools must be installed and be in one of the paths of the
  `$PATH` environment variable.

## Quickstart Guide
The chapter describes a quick way to setup baur and dependencies for
experimenting with it locally. It's not suitable for running baur in production.


1. Install baur by either:
  - downloading and installing a release archive from https://github.com/simplesurance/baur/releases or
  - running `go get -u https://github.com/simplesurance/baur` to install the
    latest development version
2. Start a PostgreSQL server in a docker container:
   `docker run -p 5432:5432 -e POSTGRES_DB=baur postgres:latest`
3. Create the baur tables in the PostgreSQL database:
   `baur init db postgres://postgres@localhost:5432/baur?sslmode=disable`

## Configuration
1. Setup your PostgreSQL baur database
1. Create a `baur.toml` file in the root of your repository by running
   `baur init repo` in the repository root.

2. Adapt the configuration files to your needs:
   - Add paths containing your applications to the `application_dirs` parameter.
   - set the `command` parameter in the `Build` section to the command that
     should be run to build your application directories
   - set the `postgresql_url` to a valid connection string to a database that
     will store information about builds, it's inputs and outputs.
     If you do not want to store the password of your database user in the
     `baur.toml` file. You can also put it in a `~/.pgpass` file
    (https://www.postgresql.org/docs/9.3/static/libpq-pgpass.html).
   - create the tables in the database by running the SQL-script
     `storage/postgres/migrations/0001.up.sql`

2. Run `baur init app` in your application directories to create an `.app.toml`
   file.
   Every application that is build via `baur` must have an `.app.toml` file.

3. Specify in your `.app.toml` files the inputs and outputs of builds.

### Application configs (`app.toml`)
#### Build Inputs
To enable baur to reliably detect if an application needs to be rebuild, it
tracks all influencing factors as build inputs.
Things that can change the output of an build can be e.g.:
- build flags change
- source file change,
- the build tools change (e.g. update to a newer gcc version),
- a docker image changes that is used to build the application

It's important that the list of build inputs and outputs is complete. Otherwise
it happens that baur won't rebuild an application despite it changed.

Build Inputs must be configured per application in the `app.toml` file with the
following directives

##### `[Build.Input.GitFiles]`
It's the preferred way to specify input files:
- it's faster for large directories then `[Build.Input.Files]`,
- it ignores untracked files like temporary build files that are in the
    repository and probably not affect the build result,

The baur repository has to be part of a checked out git repository to work.

It's `paths` parameter accepts a list git path patterns that are relative to the
application directory.
It only matches files that are tracked by the git repository, untracked files
are ignored. Modified tracked files are considered.

##### `[Build.Input.Files]`
The section has a `paths` parameter that accepts a list of glob paths to source files.

To make it easier to track changes in the build environment it's advised to
build application in docker containers and define the docker image as build
input.

##### `[[Build.Input.DockerImages]]`
Specifies a docker image as build input. This can be for example the docker
image in that the application is build.
The docker image is specified by it's manifest digest.
The manifest digest for a docker image can be retrieved with
`docker images --digests` or `docker inspect`

#### `[Build.Input.GolangSources]`
Allows to add Golang applications as inputs.
The `paths` parameters take a list of paths to directories relative to the
application directory.
In every directory `.go` files are discovered and the files they depend on.
Imports in the `.go` files are evaluated, resolved to files and tracked as build
inputs.
Go test files and imports belong to the Golang stdlib are ignored.

To be able to resolve the imports either the `GOPATH` environment variable must
be set correctly or alternatively the `go_path` parameter in the config section
must be set to the `GOPATH`. The `go_path` expects a path relative to the
application directory.

### Build Outputs
Build outputs are the results that are produced by a build. They can be
described in the `[Build.Output]` section.
Baur supports to upload build results to S3 and docker images to a remote docker
repository.
Authentication information for output repositories are either read from the
docker credentials helper or, if set, read from
`DOCKER_USERNAME` and `DOCKER_PASSWORD` environment variables.
`DOCKER_PASSWORD` can be the cleartext password or a valid authentication token.

S3 configuration parameters are the same then for the aws CLI tool.
See https://docs.aws.amazon.com/cli/latest/userguide/cli-environment.html.

The `dest_file` parameter in the `[Build.Output.File.S3Upload]` sections and the
`tag` parameter of `[Build.Output.DockerImage.RegistryUpload]` sections support
variables in their values.
The variables are replaced by baur during a run.

The following variables are supported:
- `$APPNAME` - is replaced with the name of the application
- `$UUID` - is replaced with a generated UUID
- `$GITCOMMIT` - is replaced with the current Git commit ID.
                 The `.baur.toml` file must be part of a git repository and the
                 `git` command must be in one of the directories in the `$PATH`
                 environment variable.

## Examples
- List all applications in the repository with their build status:
  `baur ls apps`
- Build all applications with outstanding builds, upload their artifacts and
  records the results:
  `baur build --upload`
- Show information about an application called `currency-service`:
  `baur show app currency-service`
- Show inputs of an application called `claim-service` with their digests:
  `baur show inputs --digests claim-server`

## Commands
### `baur verify`
Verify can be used to check for inconsistencies in past builds.
It can find builds that have the same totalinputdigest but produced different
artifacts. This can either mean that the build output it not reproducible (same
inputs don't produce the same output) or that the specified Inputs are not
complete.
The command only compares the digests of file outputs. Docker container digest
always differ when the container is rebuild.

To analyze differences in build outputs the [https://diffoscope.org/](diffoscope)
tool can be handy.

## Development
### Create new Release
1. Update the version number in the `ver` file and commit the change.
2. Run `make release` to create the release `tar.xz` archives.
3. Create a new git tag (follow the instructions printed by `make release`).
4. Push the `ver` file change to the remote git repository.
5. Create a new release on github.com and upload the binaries.

## CI Status: [![CircleCI](https://circleci.com/gh/simplesurance/baur.svg?style=svg&circle-token=8bc17577e45f5246cba2e1ea199ae504c8700eb6)](https://circleci.com/gh/simplesurance/baur)


[modeline]: # ( vi:set tabstop=4 shiftwidth=4 tw=80 expandtab spell spl=en_us : )
