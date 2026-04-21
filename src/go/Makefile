DIST    ?= ../../dist
GO      ?= go
GOFLAGS ?= -trimpath
LDFLAGS ?= -s -w

.PHONY: all clean test run build upgrade install uninstall parity help

all: build					# default: build binary

clean:						# remove binary
	@rm -f $(DIST)/pylon

test:						# run go test
	@$(GO) test ./...

run:						# no-op for a CLI; use install + pylon
	@echo "use 'make install' then invoke 'pylon'"

# build: produce a release-style binary.
#   -ldflags="-s -w" strips the symbol table (-s) and DWARF debug info (-w),
#   shrinking the binary ~20-30% with no runtime cost; stack traces still
#   work but lose file/line info.
#   -trimpath removes local build-path prefixes for reproducibility and a
#   small additional size win.
build:						# build the binary to $(DIST)/pylon
	@mkdir -p $(DIST)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/pylon .

upgrade:					# update all go module dependencies
	@$(GO) get -u ./...
	@$(GO) mod tidy

install:					# install to $(GOBIN) or $(GOPATH)/bin
	@$(GO) install .

uninstall:					# remove binary from $(DIST)
	@rm -f $(DIST)/pylon

parity: build					# byte-compare ASCII output of Go vs JS on the same fixtures
	@../../scripts/pylon-parity.sh

help:						# show this message
	@printf "Usage: make -C src/go [OPTION]\n"
	@printf "\n"
	@perl -nle 'print $$& if m{^[\w-]+:.*?#.*$$}' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?#"} {printf "    %-18s %s\n", $$1, $$2}'
