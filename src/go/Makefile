DIST ?= ../../dist
GO   ?= go

.PHONY: all clean test run build upgrade install uninstall help

all: build					# default: build binary

clean:						# remove binary
	@rm -f $(DIST)/pylon

test:						# run go test
	@$(GO) test ./...

run:						# no-op for a CLI; use install + pylon
	@echo "use 'make install' then invoke 'pylon'"

build:						# build the binary to $(DIST)/pylon
	@mkdir -p $(DIST)
	@$(GO) build -o $(DIST)/pylon .

upgrade:					# update all go module dependencies
	@$(GO) get -u ./...
	@$(GO) mod tidy

install:					# install to $(GOBIN) or $(GOPATH)/bin
	@$(GO) install .

uninstall:					# remove binary from $(DIST)
	@rm -f $(DIST)/pylon

help:						# show this message
	@printf "Usage: make -C src/go [OPTION]\n"
	@printf "\n"
	@perl -nle 'print $$& if m{^[\w-]+:.*?#.*$$}' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?#"} {printf "    %-18s %s\n", $$1, $$2}'
