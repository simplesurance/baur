# vi:set tabstop=8 sts=8 shiftwidth=8 noexpandtab tw=80:

GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_DIRTY := $(if $(shell git diff-files),wip)
VERSION := $(shell cat ver)
LDFLAGS := "-X github.com/simplesurance/baur/version.GitCommit=$(GIT_COMMIT) \
	    -X github.com/simplesurance/baur/version.Version=$(VERSION) \
	    -X github.com/simplesurance/baur/version.Appendix=$(GIT_DIRTY)"
BUILDFLAGS := -mod=vendor
TARFLAGS := --sort=name --mtime='2018-01-01 00:00:00' --owner=0 --group=0 --numeric-owner

default: all

all: baur

.PHONY: baur
baur: cmd/baur/main.go
	$(info * building $@)
	@CGO_ENABLED=0 go build $(BUILDFLAGS) -ldflags=$(LDFLAGS) -o "$@"  $<

.PHONY: dist/darwin_amd64/baur
dist/darwin_amd64/baur:
	$(info * building $@)
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
		$(BUILDFLAGS) -ldflags=$(LDFLAGS) -o "$@" cmd/baur/main.go
	$(info * creating $(@D)/baur-darwin_amd64-$(VERSION).tar.xz)
	@tar $(TARFLAGS) -C $(@D) -cJf $(@D)/baur-darwin_amd64-$(VERSION).tar.xz $(@F)

.PHONY: dist/linux_amd64/baur
dist/linux_amd64/baur:
	$(info * building $@)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		$(BUILDFLAGS) -ldflags=$(LDFLAGS) -o "$@" cmd/baur/main.go
	$(info * creating $(@D)/baur-linux_amd64-$(VERSION).tar.xz)
	@tar $(TARFLAGS) -C $(@D) -cJf $(@D)/baur-linux_amd64-$(VERSION).tar.xz $(@F)

.PHONY: dirty_worktree_check
dirty_worktree_check:
	@if ! git diff-files --quiet || git ls-files --other --directory --exclude-standard | grep ".*" > /dev/null ; then \
		echo "remove untracked files and changed files in repository before creating a release, see 'git status'"; \
		exit 1; \
		fi

.PHONY: release
release: clean dirty_worktree_check dist/linux_amd64/baur dist/darwin_amd64/baur
	@echo
	@echo next steps:
	@echo - git tag v$(VERSION)
	@echo - git push --tags
	@echo - upload $(ls dist/*/*.tar.xz) files

.PHONY: check
check:
	$(info * running static code checks)
	@gometalinter ./...

.PHONY: clean
clean:
	@rm -rf baur dist/

.PHONY: test
test:
	go test $(BUILDFLAGS) -race -v ./...
