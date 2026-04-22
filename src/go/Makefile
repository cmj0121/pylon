DIST    ?= ../../dist
GO      ?= go
GOFLAGS ?= -trimpath
LDFLAGS ?= -s -w

# VERSION is embedded into the pylon / pylon-lsp binaries via
# -ldflags="-X main.Version=...". `git describe` yields
# v<tag>-<commits>-g<sha>[-dirty]; unreleased worktrees without any
# tag fall back to the short SHA, and environments without git at all
# get the literal "dev" marker.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
VERSION_LDFLAGS := -X main.Version=$(VERSION)

PYLON          := $(DIST)/pylon
PYLON_LSP      := $(DIST)/pylon-lsp
PYLON_DUMP_DIAG := $(DIST)/pylon-dump-diagnostics

.PHONY: all clean test run build upgrade install uninstall parity parity-diagnostics help

all: build					# default: build binaries

clean:						# remove binaries
	@rm -f $(PYLON) $(PYLON_LSP) $(PYLON_DUMP_DIAG)

test:						# run go test
	@$(GO) test ./...

run:						# no-op for a CLI; use install + pylon
	@echo "use 'make install' then invoke 'pylon'"

# build: produce release-style binaries.
#   -ldflags="-s -w" strips the symbol table (-s) and DWARF debug info (-w),
#   shrinking the binary ~20-30% with no runtime cost; stack traces still
#   work but lose file/line info.
#   -trimpath removes local build-path prefixes for reproducibility and a
#   small additional size win.
build: $(PYLON) $(PYLON_LSP) $(PYLON_DUMP_DIAG)	# build all three binaries to $(DIST)/

$(PYLON):
	@mkdir -p $(DIST)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS) $(VERSION_LDFLAGS)" -o $(PYLON) ./cmd/pylon

$(PYLON_LSP):
	@mkdir -p $(DIST)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS) $(VERSION_LDFLAGS)" -o $(PYLON_LSP) ./cmd/pylon-lsp

$(PYLON_DUMP_DIAG):
	@mkdir -p $(DIST)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(PYLON_DUMP_DIAG) ./cmd/pylon-dump-diagnostics

upgrade:					# update all go module dependencies
	@$(GO) get -u ./...
	@$(GO) mod tidy

install:					# install binaries to $(GOBIN) or $(GOPATH)/bin
	@$(GO) install ./cmd/pylon ./cmd/pylon-lsp

uninstall:					# remove binaries from $(DIST)
	@rm -f $(PYLON) $(PYLON_LSP) $(PYLON_DUMP_DIAG)

parity: $(PYLON)				# byte-compare ASCII output of Go vs JS on the same fixtures
	@../../scripts/pylon-parity.sh

parity-diagnostics: $(PYLON_DUMP_DIAG)		# diff JS and Go diagnostic sets over the bad-input corpus
	@../../scripts/pylon-diagnostics-parity.sh

help:						# show this message
	@printf "Usage: make -C src/go [OPTION]\n"
	@printf "\n"
	@perl -nle 'print $$& if m{^[\w-]+:.*?#.*$$}' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?#"} {printf "    %-22s %s\n", $$1, $$2}'
