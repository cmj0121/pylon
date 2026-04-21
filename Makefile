SUBDIR := src/js contrib/vim

.PHONY: all clean test run build upgrade install uninstall help $(SUBDIR)

all: $(SUBDIR) 			# default action
	@[ -f .git/hooks/pre-commit ] || pre-commit install --install-hooks
	@git config commit.template .git-commit-template

clean: $(SUBDIR)		# clean-up environment
	@find . -name '*.sw[po]' -delete

test: $(SUBDIR)			# run test

run: $(SUBDIR)			# run in the local environment

build: $(SUBDIR)		# build the binary/library

upgrade:				# upgrade all the necessary packages
	pre-commit autoupdate

install: $(SUBDIR)		# install everything into the local environment

uninstall: $(SUBDIR)	# uninstall everything from the local environment

help:					# show this message
	@printf "Usage: make [OPTION]\n"
	@printf "\n"
	@perl -nle 'print $$& if m{^[\w-]+:.*?#.*$$}' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?#"} {printf "    %-18s %s\n", $$1, $$2}'

$(SUBDIR):
	$(MAKE) -C $@ $(MAKECMDGOALS)
