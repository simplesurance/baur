# Commandline

## General Parameters
- `-v/--verbose`
- `--cpu-prof`
- `--version`


## `baur repo`
```
baur repo init
baur repo show
```

## `baur app`
- ```
  baur app init [<APPNAME>]
  ```
  Create a new application config, in the current directory, the default
  `<APPNAME>` is the name of the current directory,
  `<APPNAME>` can be passed to specify a different appname


- ```
  baur app build [-n/--no-upload] [-i/--ignore-errors] [-t/--timeout=<DURATION>] [-f/--force] [<APP>|<PATH>]...
  ```
  Builds and uploads by default all outstanding applications in the repository
  and abort on the first build error

- ```
  baur app show [<APP>|<PATH>] <-- no argument, shows app in current directory
  ```
  Shows informations about the application configuration (name, input dirs,
  output dirs, etc)


- ```
  baur apps ls [--csv] [-q] [--status=<BUILD-STATUS>] [--sort=<FIELD>-<SORT-ORDER>] --abs-path
  <FIELD> is one of: app, path, build-id, status
  ```
  Shows informations about applications
  If `--quiet` Is passed only application names are shown


## `baur builds`
- ```
  baur build show <BUILD-ID>
  ``
  Shows information about a recorded build

- ```
  baur build ls [--csv] [-q/--quiet] [--max=<COUNT>] [--fields=<FIELD>...] [--<FIELD>=<VALUE>]... [--sort-by=<FIELD>...] [--sort-order=asc|desc] [<APP>|<PATH>]
  FIELD is one of: application-name, number of builds, build-start-date,
			build-end-date, build-duration, artifact-size, upload-duration,
  ```
  List informations about builds
  If no argument is passed it builds of the  app in the current directory
  if `-q` is passed only build-ids are printed


## `baur inputs`
- ```
  baur inputs diff <BUILDID> <BUILDID>
  ```
  Compares the input digests of two builds, it prints which outputs differ
  (similiar to diff view)

- ```
   baur inputs show [--csv] --digests  <APP>|<BUILDID>|<PATH>
   baur apps inputs  [--csv] --digests  <APP|PATH> (can be an alias of inputs show)
   baur builds inputs  [--csv] --digests  <BUILDID> (can be an alias of inputs show
  ```
  list inputs of an app or, build
