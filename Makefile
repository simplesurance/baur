GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_DIRTY := $(if $(shell git diff-files),wip)
LDFLAGS := "-X github.com/simplesurance/baur/v3/internal/version.GitCommit=$(GIT_COMMIT) \
	    -X github.com/simplesurance/baur/v3/internal/version.Appendix=$(GIT_DIRTY)"
BUILDFLAGS := -trimpath -ldflags=$(LDFLAGS)
export GO111MODULE=on
export GOFLAGS=-mod=vendor

default: all

all: baur

.PHONY: baur
baur: cmd/baur/main.go
	$(info * building $@)
	@CGO_ENABLED=0 go build $(BUILDFLAGS) -o "$@"  $<

.PHONY: check
check:
	$(info * running static code checks)
	@golangci-lint run

.PHONY: clean
clean:
	@rm -rf baur dist/

.PHONY: test
test:
	go test -race ./...

.PHONY: dbtest
dbtest:
	go test -race -tags=dbtest,s3test ./...
