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


Baur makes certain Assumptions:
- an application directory only contains one application,
- an application can be build by running a single command,
  a build has to produce 1 or more build artifacts


## Build
To build the application run `make`

## Dependencies
- To make use of git specify functionality like using the `$GITCOMMIT` variable
    and `[GitSourceFiles]` directive in config files,  the git commandline tools
    must be installed and findable via the `$PATH` environment variable.

## Configuration
1. Setup your PostgreSQL baur database
1. Create a `baur.toml` file in the root of your repository by running
   `baur init` in the repository root.

2. Adapt the configuration files to your needs:
   - Add paths containing your applications to the `application_dirs` parameter.
   - set the build_command to a command that when it's run in your application
	 directories, produces build artifacts like a docker container or a tar
	 archives.
   - set the `postgresql_url` to a valid connection string to a database that will
    store informations about builds and aritfacts.
    If you do not want to store the password of your database user in the
    `baur.toml` file. You can also put in your `.pgpass` file
    (https://www.postgresql.org/docs/9.3/static/libpq-pgpass.html).
   - create the tables in the database by running the SQL-script
     `storage/postgres/migrations/0001.up.sql`

2. Run `baur appinit` in your application directories to create an `.app.toml`
   file.
   Every application that is build via `baur` must have an `.app.toml` file.

3. Specify in your `.app.toml` files the artifacts that are produced by builds
   and where they should be uploaded to.
   Baur supports uploading artifacts to S3 and docker containers to
   hub.docker.com.
   Authentication information for artifact repositories are read from
   environment variables. S3 configuration parameters are the same
   then for the aws CLI tool. See
   https://docs.aws.amazon.com/cli/latest/userguide/cli-environment.html

   The credentials for the hub.docker.com registry can be specified by setting
   the `DOCKER_USERNAME` and `DOCKER_PASSWORD` environment variables.
   `DOCKER_PASSWORD` can be the cleartext password or a valid authentication
   token.

### Application configs (`app.toml`)
The `dest_file` parameter in the `[[DockerArtifact]]` sections and the `tags`
parameter of `[[S3Artifact]]` sections support variables in their values.
The variables are replaced by baur during a run.

The following variables are supported:
- `$APPNAME` - is replaced with the name of the application
- `$UUID` - is replaced with a generated UUID
- `$GITCOMMIT` - is replaced with the current Git commit ID.
                 The `.baur.toml` file must be part of a git repository and the
                 `git` command must be in one of the directories in the `$PATH`
                 environment variable.

#### Application Sources
To enable baur to reliably detect if an application needs to be rebuild, it
tracks all influencing factor of a build. This can be:
The artifacts that an application build produces change when:
- build flags change
- the source files change,
- the build tools change (e.g. update to a newer gcc version).

It's important that the list is complete. Otherwise it happens that baur won't
rebuild an application despite it changed.

Those sources must be configured per application in the `app.toml` file with the
following directives

##### `[GitSourceFiles]`
It's the preferred way to specify sources by files:
- it's faster for large directories then `[SourceFiles]`,
- it ignores untracked files like temporary build files that are in the
    repository and probably not affect the build result,

The baur repository has to be part of a checked out git repository to work.

It's `paths` parameter accepts a list git path patterns that are relative to the
application directory.
It only matches files that are tracked by the git repository. Untracked files
are ignored. Modified tracked files are considered.

##### `[SourceFiles]`
Has a `paths` parameter that accepts a list of glob paths to source files.

To make it easier to track changes in the build environment it's advised to
build application in docker containers and define the docker image as Source
(not supported yet).

(depending if an application needs to be rebuild because it sources changed is
not implemented yet)

##### `[[DockerSource]]`
Specifies a docker image as build input. This can be for example the docker
image in that the application is build.
It's identified by manifest digest to ensure that it is unambiguous.
The digest for a docker image can be retrieved by e.g.
`docker images --digests`.

## Examples
- List all applications in the repository:
  `baur ls`
- Build all applications, upload their artifacts and records the results:
  `baur build --upload all`
- Show informations about an application called `currency-service`:
  `baur show currency-service`
- Show source files of an application called `claim-service`:
  `baur sources claim-server`

## Development
### Create new Release
1. Update the version number in the `ver` file and commit the change.
2. Run `make release` to create the release `tar.xz` archives.
3. Create a new git tag (follow the instructions printed by `make release`).
4. Push the `ver` file change to the remote git repository.
5. Create a new release on github.com and upload the binaries.

[modeline]: # ( vi:set tabstop=4 shiftwidth=4 tw=80 expandtab spell spl=en : )
