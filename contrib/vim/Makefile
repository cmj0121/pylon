VIM_PACK ?= $(HOME)/.config/nvim/pack/pylon/start/pylon

.PHONY: all clean test run build upgrade install uninstall help $(SUBDIR)

all: $(SUBDIR) 		# default action

clean: $(SUBDIR)	# clean-up environment

test: $(SUBDIR)		# run test

run: $(SUBDIR)		# run in the local environment

build: $(SUBDIR)	# build the binary/library

upgrade:			# upgrade all the necessary packages

install:			# copy syntax/ and ftdetect/ into $(VIM_PACK)
	@mkdir -p $(VIM_PACK)
	@cp -r syntax ftdetect $(VIM_PACK)/

uninstall:			# remove $(VIM_PACK) entirely
	@rm -rf $(VIM_PACK)

help:				# show this message
	@printf "Usage: make [OPTION]\n"
	@printf "\n"
	@perl -nle 'print $$& if m{^[\w-]+:.*?#.*$$}' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?#"} {printf "    %-18s %s\n", $$1, $$2}'

$(SUBDIR):
	$(MAKE) -C $@ $(MAKECMDGOALS)
