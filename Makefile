#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.
#

#
# Copyright 2020 Joyent, Inc.
#

NAME = triton-shim



#
# Makefile.defs defines variables used as part of the build process.
# Ensure we have the eng submodule before attempting to include it.
#
ENGBLD_REQUIRE          := $(shell git submodule update --init deps/eng)
include ./deps/eng/tools/mk/Makefile.defs
TOP ?= $(error Unable to access eng.git submodule Makefiles.)

#
# Configuration used by Makefile.defs and Makefile.targ to generate
# "check" and "docs" targets.
#
BASH_FILES =		tools/check-copyright
DOC_FILES =		README.md docs/README.md

#
# Configuration used by Makefile.smf.defs to generate "check" and "all" targets
# for SMF manifest files.
#
SMF_MANIFESTS_IN =	smf/manifests/triton-shim.xml.in
include ./deps/eng/tools/mk/Makefile.smf.defs


#
# If a project includes some components written in the Go language, the Go
# toolchain will need to be available on the build machine.  At present, the
# Makefile library only handles obtaining a toolchain for SmartOS systems.
#
ifeq ($(shell uname -s),SunOS)
	GO_PREBUILT_VERSION =	1.14
	GO_TARGETS =		$(STAMP_GO_TOOLCHAIN)
	include ./deps/eng/tools/mk/Makefile.go_prebuilt.defs
else
	GO = $(shell which go)
	GOOS = $(shell $(GO) env GOOS)
	GOARCH = $(shell $(GO) env GOARCH)
endif

GO_TEST_DIRECTORIES =	./actions ./api ./server

#
# Repo-specific targets
#
.PHONY: all
all: $(SMF_MANIFESTS) $(GO_TARGETS) | $(REPO_DEPS)

.PHONY: release
release:
	echo "Do work here: tag the release. We're gonna use git submodules at first pass"

.PHONY: test
test: $(STAMP_GO_TOOLCHAIN)
	@$(GO) version
	$(GO) test $(GO_TEST_DIRECTORIES) -count=1

#
# Target definitions.  This is where we include the target Makefiles for
# the "defs" Makefiles we included above.
#

include ./deps/eng/tools/mk/Makefile.deps

ifeq ($(shell uname -s),SunOS)
	include ./deps/eng/tools/mk/Makefile.go_prebuilt.targ
endif


include ./deps/eng/tools/mk/Makefile.smf.targ
include ./deps/eng/tools/mk/Makefile.targ
