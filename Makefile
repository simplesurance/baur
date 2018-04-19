# vi:set tabstop=8 sts=8 shiftwidth=8 noexpandtab tw=80:
#
GIT_DESCRIBE := $(shell git describe --always --dirty --abbrev)
LDFLAGS := "-X github.com/simplesurance/sisubuild/version.GitDescribe=$(GIT_DESCRIBE)"

default: all

all: sb

.PHONY: sb
sb:
	@echo "* building $@"
	@CGO_ENABLED=0 go build -ldflags=$(LDFLAGS) -o "$@"

.PHONY: check
check:
	@echo "* running static code checks"
	@gometalinter \
		--deadline 10m \
		--vendor \
		--sort="path" \
		--aggregate \
		--enable-gc \
		--disable-all \
		--enable goimports \
		--enable misspell \
		--enable vet \
		--enable deadcode \
		--enable varcheck \
		--enable ineffassign \
		--enable structcheck \
		--enable unconvert \
		--enable gofmt \
		--enable unused \
		./...
