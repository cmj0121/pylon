PORT ?= 3333

.PHONY: all clean test run build help

all: build			# default action

clean:				# clean-up environment
	@rm -rf dist/

test:				# run test
	@echo "no tests yet"

run:				# serve the demo at http://localhost:$(PORT)
	@echo "serving src/js at http://localhost:$(PORT)/ -- press Ctrl-C to stop"
	@python3 -m http.server $(PORT)

build:				# build the library (no-op: plain JS, no bundler)
	@echo "pylon.js is plain JS -- no build step"

help:				# show this message
	@printf "Usage: make -C src/js [OPTION]\n"
	@printf "\n"
	@perl -nle 'print $$& if m{^[\w-]+:.*?#.*$$}' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?#"} {printf "    %-18s %s\n", $$1, $$2}'
