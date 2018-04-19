# sisubuild

sisubuild manages builds and artifacts in mono repositories.

When Git repositories contain only a single applications the CI/CD jobs can be
triggered on every new commit.
Monorepositories can contain hundreds of apps and it becomes inefficient and
slow to building, test and deploy all applications on every commit.
A solution is required to detect which applications have changed and run CI/CD
tasks only for those.

sisubuild will implement:
- discovery of applications in a repository,
- management to store and retrieve build artifacts for applications,
- detect if build artifacts for an application version already exist or if it's
  need to be build


[modeline]: # ( vi:set tabstop=4 shiftwidth=4 tw=80 expandtab spell spl=en : )

## Build
To build the application run `make`

## Configuration
1. Create a `sisubuild.toml` file in the root of your repository by running 
   `sb init` in the repository root.
   Adapt the configuration files to your needs. Add paths containing your
   applications to the `application_dirs` parameter.



2. Create `.app.toml` files in all directories that contain applications. The
   file must have a `name` parameter. Example:

   ```
   name = "i18n-service
   ```


## Examples
- List all applications in the repository:
  `sb ls`
