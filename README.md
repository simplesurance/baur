# baur

baur manages builds and artifacts in mono repositories.

When Git repositories contain multiple single apps, the CI/CD jobs can be
triggered on every new commit.
Monorepositories can contain hundreds of apps and it becomes inefficient and
slow to build, test and deploy all applications on every commit.
The solution is to detect app changes and run CI/CD only where needed.

baur implements:
- discovery of applications in a repository,
- storage and retrieval of build artifacts for individual applications,
- existing artifact detection per app state (or is it time for a new build?)


baur makes the following assumptions:
- an application directory only contains one application,
- an application can be build by running a single command,
- a build produces 1 or more build artifacts



## Build
To build the application run `make`

## Configuration
1. Create a `baur.toml` file in the root of your repository by running
   `baur init` in the repository root.
   Adapt the configuration files to your needs:

   - add paths containing your applications to the `application_dirs` parameter.
   - set `build_command` to the command that, when it's run in your application
	 directories, produces build artifacts (like a docker container or a tar
	 archive).

2. Run `baur appinit` in your application directories to create a `.app.toml`
   file. Every application that is build via `baur` must have an `.app.toml`
   file.


## Examples
- List all applications in the repository:
  `baur ls`
- Build all applications in the repository:
  `baur build all`

[modeline]: # ( vi:set tabstop=4 shiftwidth=4 tw=80 expandtab spell spl=en : )


## Development

### Create new Release
1. Update the version number in the `ver` file and commit the change.
2. Run `make release` to create the release `tar.xz` archives.
3. Create a new git tag (follow the instructions printed by `make release`).
4. Push the `ver` file change to the remote git repository.
5. Create a new release on github.com and upload the binaries.
