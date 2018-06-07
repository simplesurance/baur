- `sisubuild test [--record] [--force] [<application-name>]`
  - runs application tests
- `sisubuild fetch <application-name> <build-id>`
- `sisubuild artifacts-ls [--builds=[<number>|all] [--verbose] <application-name> [<commit-id>]`
  - list informations about artifacts in database
  - `--builds` specifies the max number of builds per application to show, default
    is 3
- `sisubuild --update` updates the sisubuild binary
- `sisubuild artifact-exist` <application-name>

[modeline]: # ( vi:set tabstop=4 ft=markdown shiftwidth=4 tw=80 expandtab spell spl=en : )
