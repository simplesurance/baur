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

## Configuration
1. Create a `baur.toml` file in the root of your repository by running
   `baur init` in the repository root.
   Adapt the configuration files to your needs:

   - Add paths containing your applications to the `application_dirs` parameter.
   - set the build_command to a command that when it's run in your application
	 directories, produces build artifacts like a docker container or a tar
	 archives.

2. Run `baur appinit` in your application directories to create an `.app.toml`
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
