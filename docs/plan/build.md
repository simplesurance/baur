Console Mock
------------
```
sisubuild build [--upload] [--force] [--record] [--verbose] <application-name>
$ sisubuild build --upload --record claim-service
* hashing source files... done (3s)
=> source hash: df285ab34ad10d8b641e65f39fa11a7d5b44571a37f94314debbfe7233021755
* building go/code/claim... done (22s)
=> go/code/claim/dist/claim-service.tar.xz (4.5M)
* uploading artifacts
=> go/code/claim/dist/claim-service.tar.xz... done (1.23MB/s)
* recording build... done
```

If exist:
```
* hashing source files... done (3s)
=> build exist, artifact: s3://simplesurance/apps/claim_service-df512sn12l.tar.xz         
```

Sources, Artifacts
------------------

A build transforms 0-n input files via a build command to 1-n artifacts.
```
SOURCES -> BUILD_CMD -> ARTIFACTS
```

The sources of an application needs to be specified. They can be either be
specified by a list of files/directories, determined by godep or a PHP
dependency resolver.

If an application can have multiple Build Commands that create different
artifacts:
- the tools to determine the sources would need to have arguments, to specify
  that e.g. the depsources for go bin X is Build A, and go-bin Y is build B,
- it would get more complicated to represent everything (app has multiple builds
  has multiple sources, etc)

If an application can only have a single Build Command but produces multiple
artifacts from different sources, it might happen that artifacts are unnecessary
build.

To keep it simple baur only supports one Build description per application for
the beginning.
If a source of an application change all it's artifact should change to be
efficient.

[modeline]: # ( vi:set tabstop=4 ft=markdown shiftwidth=4 tw=80 expandtab spell spl=en : )
