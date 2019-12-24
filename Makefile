# ----------------------------------------------------------------------------
# global

SHELL = /usr/bin/env bash
.DEFAULT_GOAL = test

# hack for replace all whitespace to comma
comma := ,
empty :=
space := $(empty) $(empty)

# ----------------------------------------------------------------------------
# Go

ifneq ($(shell command -v go),)
GO_PATH ?= $(shell go env GOPATH)
GO_OS   ?= $(shell go env GOOS)
GO_ARCH ?= $(shell go env GOARCH)
GO_BIN = ${CURDIR}/bin
CGO_ENABLED ?= 0

PKG := $(subst $(GO_PATH)/src/,,$(CURDIR))
GO_ALL_PKGS := $(shell go list ./... | grep -v -e '.pb.go')
GO_PKGS := $(shell go list -f '{{if and (or .GoFiles .CgoFiles) (ne .Name "main")}}{{.ImportPath}}{{end}}' ${PKG_PATH}/...)
GO_TEST_PKGS := $(shell go list -f='{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...)
GO_VENDOR_PKGS=
ifneq ($(wildcard ./vendor),)  # exist vender directory
GO_VENDOR_PKGS=$(shell go list ${GO_MOD_FLAGS} -f '{{if and (or .GoFiles .CgoFiles) (ne .Name "main")}}./vendor/{{.ImportPath}}{{end}}' ./vendor/...)
endif
endif

GO_TEST ?= go test
GO_TEST_FUNC ?= .
GO_TEST_FLAGS ?=
GO_TEST_COVERAGE_OUT ?= coverage.out
GO_BENCH_FUNC ?= .
GO_BENCH_FLAGS = -benchmem

GO_BUILD_TAGS ?=
ifeq (${CGO_ENABLED},0)
	GO_BUILD_TAGS+=osusergo netgo
endif
GO_BUILD_TAGS_STATIC=static static_build
GO_INSTALLSUFFIX_STATIC=-installsuffix 'netgo'
GO_FLAGS=-tags='$(subst $(space),$(comma),${GO_BUILD_TAGS})'

GO_GCFLAGS ?=
GO_LDFLAGS=-s -w
GO_LDFLAGS_STATIC="-extldflags=-fno-PIC -static"
GO_CHECKPTR_FLAGS=all=-d=checkptr=1 -d=checkptr=2
GO_GCFLAGS_DEBUG=all=-N -l -dwarflocationlists=true
GO_LDFLAGS_DEBUG=-compressdwarf=false
ifeq (${GO_GCFLAGS},)
	GO_FLAGS+=-gcflags='${GO_GCFLAGS}'
endif
ifeq (${GO_LDFLAGS},)
	GO_FLAGS+=-ldflags='${GO_LDFLAGS}'
endif

# ----------------------------------------------------------------------------
# defines

## target
GOPHER=""
define target
@printf "$(GOPHER)  \\x1b[1;32m$(patsubst ,$@,$(1))\\x1b[0m\\n"
endef

## tools
### $1: package import path, $2 revision
define tools
	$(call target,tools/$(@F))
	@{ \
		printf "downloadnig $(@F) ...\\n\\n" ;\
		set -e ;\
		TMP_DIR=$(shell go env TMPDIR)/tools ;\
		mkdir -p ${TMP_DIR} ;\
		cd ${TMP_DIR} ;\
		go mod init tmp > /dev/null 2>&1 ;\
		CGO_ENABLED=0 GOOS=${GO_OS} GOARCH=${GO_ARCH} GOBIN=${GO_BIN} go get -u -tags='osusergo,netgo,static,static_build' -ldflags='-s -w "-extldflags=-fno-PIC -static"' -installsuffix 'netgo' ${1}@${2} > /dev/null 2>&1 ;\
		rm -rf ${TMP_DIR} ;\
	}
endef

### $1: download URL, $2: src, $3: dest
define toools_release
	$(call target,tools/$(@F))
	@{ \
		mkdir -p ${GO_BIN} ;\
		printf "downloadnig $(@F) ...\\n\\n" ;\
		set -e ;\
		TMP_DIR=$(shell go env TMPDIR)/$(@F) ;\
		mkdir -p ${TMP_DIR} ;\
		curl -sSL ${1} | tar -xz -C ${TMP_DIR} ;\
		mv ${2} ${3} ;\
		rm -rf ${TMP_DIR} ;\
	}
endef

# ----------------------------------------------------------------------------
# targets

##@ test, bench, coverage

.PHONY: test
test: CGO_ENABLED=1  # needs race test
test: GO_LDFLAGS+=${GO_LDFLAGS_STATIC}
test: GO_BUILD_TAGS+=${GO_BUILDTAGS_STATIC}
test: GO_FLAGS+=${GO_INSTALLSUFFIX_STATIC}
test:  ## Runs package test including race condition.
	$(call target)
	CGO_ENABLED=$(CGO_ENABLED) $(GO_TEST) -v -race $(strip $(GO_FLAGS)) -run=$(GO_TEST_FUNC) $(GO_TEST_PKGS)

.PHONY: bench
test: GO_LDFLAGS+=${GO_LDFLAGS_STATIC}
test: GO_BUILD_TAGS+=${GO_BUILDTAGS_STATIC}
test: GO_FLAGS+=${GO_INSTALLSUFFIX_STATIC}
bench:  ## Take a package benchmark.
	$(call target)
	@CGO_ENABLED=$(CGO_ENABLED) $(GO_TEST) -v $(strip $(GO_FLAGS)) -run='^$$' -bench=$(GO_BENCH_FUNC) -benchmem $(GO_TEST_PKGS)

.PHONY: coverage
coverage: GO_LDFLAGS+=${GO_LDFLAGS_STATIC}
coverage: GO_BUILDTAGS+=${GO_BUILDTAGS_STATIC}
coverage: GO_FLAGS+=${GO_INSTALLSUFFIX_STATIC}
coverage:  ## Takes packages test coverage.
	$(call target)
	CGO_ENABLED=$(CGO_ENABLED) $(GO_TEST) -v $(strip $(GO_TEST_FLAGS)) $(strip $(GO_FLAGS)) -covermode=atomic -coverpkg=./... -coverprofile=${GO_TEST_COVERAGE_OUT} $(GO_PKGS)

tools/go-junit-report:  # go get 'go-junit-report' binary
tools/go-junit-report: ${GO_BIN}/go-junit-report
${GO_BIN}/go-junit-report:
ifeq (, $(shell test -f ./bin/$(@F)))
	$(call tools,github.com/jstemmer/go-junit-report,master)
GO_JUNIT_REPORT=${GO_BIN}/go-junit-report
endif

.PHONY: ci/coverage
ci/coverage: tools/go-junit-report
ci/coverage: GO_LDFLAGS+=${GO_LDFLAGS_STATIC}
ci/coverage: GO_BUILDTAGS+=${GO_BUILDTAGS_STATIC}
ci/coverage: GO_FLAGS+=${GO_INSTALLSUFFIX_STATIC}
ci/coverage:  ## Takes packages test coverage, and output coverage results to CI artifacts.
	$(call target)
	@mkdir -p /tmp/artifacts /tmp/test-results
	CGO_ENABLED=$(CGO_ENABLED) $(GO_TEST) -v $(strip $(GO_TEST_FLAGS)) $(strip $(GO_FLAGS)) -covermode=atomic -coverpkg=./... -coverprofile=${GO_TEST_COVERAGE_OUT} $(GO_PKGS) 2>&1 | tee /dev/stderr | ${GO_JUNIT_REPORT} -set-exit-code > /tmp/test-results/junit.xml
	@if [[ -f "${GO_TEST_COVERAGE_OUT}" ]]; then go tool cover -html=${GO_TEST_COVERAGE_OUT} -o $(dir GO_TEST_COVERAGE_OUT)/coverage.html; fi


##@ fmt, lint

.PHONY: lint
fmt: fmt/gofumports  ## Run format.

tools/gofumports:  # go get 'gofumports' binary
tools/gofumports: ${GO_BIN}/gofumports
${GO_BIN}/gofumports:
ifeq (, $(shell test -f ./bin/$(@F)))
	$(call tools,mvdan.cc/gofumpt/gofumports,master)
GOFUMPORTS=${GO_BIN}/gofumports
endif

.PHONY: fmt/gofumports
fmt/gofumports: tools/gofumports
fmt/gofumports: GO_PKG_DIRS+=${CMD}
fmt/gofumports:
	${GOFUMPORTS} -w -local=${PKG_PATH} ${GO_PKG_DIRS}

.PHONY: lint
lint: lint/golangci-lint  ## Run all linters.

tools/golangci-lint:  # go get 'golangci-lint' binary
tools/golangci-lint: ${GO_BIN}/golangci-lint
${GO_BIN}/golangci-lint:
ifeq (, $(shell test -f ./bin/$(@F)))
	$(call tools,github.com/golangci/golangci-lint/cmd/golangci-lint,master)
GOLANGCI_LINT=${GO_BIN}/golangci-lint
endif

.PHONY: lint/golangci-lint
lint/golangci-lint: tools/golangci-lint .golangci.yml  ## Run golangci-lint.
	$(call target)
	@${GOLANGCI_LINT} run ./...


##@ mod

.PHONY: mod/tidy
mod/tidy:  ## Makes sure go.mod matches the source code in the module.
	$(call target)
	@go mod tidy -v

.PHONY: mod/vendor
mod/vendor: mod/tidy  ## Resets the module's vendor directory and fetch all modules packages.
	$(call target)
	@go mod vendor -v

.PHONY: mod/graph
mod/graph:  ## Prints the module requirement graph with replacements applied.
	$(call target)
	@go mod graph

.PHONY: mod/install
mod/install: mod/tidy mod/vendor
mod/install:  ## Install the module vendor package as an object file.
	$(call target)
	@go install -v $(GO_VENDOR_PKGS) || GO111MODULE=off go install -mod=vendor -v $(GO_VENDOR_PKGS)

.PHONY: mod/update
mod/update: mod/tidy mod/vendor mod/install  ## Updates all of vendor packages.
	@go mod edit -go 1.13

.PHONY: mod
mod: mod/tidy mod/vendor mod/install
mod:  ## Updates the vendoring directory using go mod.
	@go mod edit -go 1.13


##@ clean

.PHONY: clean
clean:  ## Cleanups binaries and extra files in the package.
	$(call target)
	@rm -rf ./bin *.out *.test *.prof trace.log


##@ boilerplate

.PHONY: boilerplate/go/%
boilerplate/go/%: BOILERPLATE_PKG_DIR=$(shell printf $@ | cut -d'/' -f3- | rev | cut -d'/' -f2- | rev | awk -F. '{print $$1}')
boilerplate/go/%: BOILERPLATE_PKG_NAME=$(if $(findstring .go,$(suffix $(BOILERPLATE_PKG_DIR))),$(basename ${@F}),$(shell printf $@ | rev | cut -d/ -f2 | rev))
boilerplate/go/%: hack/boilerplate/boilerplate.go.txt
boilerplate/go/%:  ## Creates a go file based on boilerplate.go.txt in % location.
	@if [ -n ${BOILERPLATE_PKG_DIR} ] && [ ! -d ${BOILERPLATE_PKG_DIR} ]; then mkdir -p ${BOILERPLATE_PKG_DIR}; fi
	@if [[ ${@F} == *'.go'* ]] || [[ ${BOILERPLATE_PKG_DIR} == *'cmd'* ]] || [ -z ${BOILERPLATE_PKG_DIR} ]; then \
		cat hack/boilerplate/boilerplate.go.txt <(printf "\npackage $(basename ${@F})\\n") > $*; \
		else \
		cat hack/boilerplate/boilerplate.go.txt <(printf "\npackage ${BOILERPLATE_PKG_NAME}\\n") > $*; \
		fi
	@sed -i "s|YEAR|$(shell date '+%Y')|g" $*


##@ miscellaneous

.PHONY: TODO
TODO:  ## Print the all of (TODO|BUG|XXX|FIXME|NOTE) in packages.
	@rg -e '(TODO|BUG|XXX|FIXME|NOTE)(\(.+\):|:)' --follow --hidden --glob='!.git' --glob='!vendor' --glob='!internal' --glob='!Makefile' --glob='!snippets' --glob='!indent'


##@ help

.PHONY: help
help:  ## Show make target help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[33m<target>\033[0m\n"} /^[a-zA-Z_0-9\/_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' ${MAKEFILE_LIST}
