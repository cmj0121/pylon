PORT ?= 3333
ESBUILD ?= esbuild@0.24.0
DIST ?= ../../dist

.PHONY: all clean test run build help

all: build			# default action

clean:				# clean-up environment
	@rm -rf $(DIST)

test:				# run test
	@echo "no tests yet"

run:				# serve the demo at http://localhost:$(PORT)
	@echo "serving src/js at http://localhost:$(PORT)/ -- press Ctrl-C to stop"
	@python3 -m http.server $(PORT)

build:				# minify pylon.js into dist/pylon.min.js
	@mkdir -p $(DIST)
	@npx --yes $(ESBUILD) pylon.js --minify --target=es2020 --outfile=$(DIST)/pylon.min.js

help:				# show this message
	@printf "Usage: make -C src/js [OPTION]\n"
	@printf "\n"
	@perl -nle 'print $$& if m{^[\w-]+:.*?#.*$$}' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?#"} {printf "    %-18s %s\n", $$1, $$2}'
