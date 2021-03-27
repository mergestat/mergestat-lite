.PHONY: clean vet test lint test-cover bench

# default task invoked while running make
all: clean .build/libaskgit.so .build/askgit

# target to build a dynamic extension that can be loaded at runtime
.build/libaskgit.so: $(shell find . -type f -name '*.go' -o -name '*.c')
	$(call log, $(CYAN), "building $@")
	@CGO_CFLAGS="-DUSE_LIBSQLITE3" CPATH="${PWD}/pkg/sqlite" \
		go build -buildmode=c-shared -o $@ -tags="static,system_libgit2,shared" shared.go
	$(call log, $(GREEN), "built $@")

# pass these flags to linker to suppress missing symbol errors in intermediate artifacts
export CGO_LDFLAGS = -Wl,--unresolved-symbols=ignore-in-object-files
ifeq ($(shell uname -s),Darwin)
	export CGO_LDFLAGS = -Wl,-undefined,dynamic_lookup
endif

# target to compile askgit executable
.build/askgit: $(shell find . -type f -name '*.go' -o -name '*.c')
	$(call log, $(CYAN), "building $@")
	@CGO_LDFLAGS="${CGO_LDFLAGS}" CGO_CFLAGS="-DUSE_LIBSQLITE3" CPATH="${PWD}/pkg/sqlite" \
		go build -o $@ -tags="sqlite_vtable,vtable,sqlite_json1,static,system_libgit2" askgit.go
	$(call log, $(GREEN), "built $@")

# target to download latest sqlite3 amalgamation code
pkg/sqlite/sqlite3.c:
	$(call log, $(CYAN), "downloading sqlite3 amalgamation source v3.35.0")
	$(eval SQLITE_DOWNLOAD_DIR = $(shell mktemp -d))
	@curl -sSLo $(SQLITE_DOWNLOAD_DIR)/sqlite3.zip https://www.sqlite.org/2021/sqlite-amalgamation-3350000.zip
	$(call log, $(GREEN), "downloaded sqlite3 amalgamation source v3.35.0")
	$(call log, $(CYAN), "unzipping to $(SQLITE_DOWNLOAD_DIR)")
	@(cd $(SQLITE_DOWNLOAD_DIR) && unzip sqlite3.zip > /dev/null)
	@-rm $(SQLITE_DOWNLOAD_DIR)/sqlite-amalgamation-3350000/shell.c
	$(call log, $(CYAN), "moving to pkg/sqlite")
	@mv $(SQLITE_DOWNLOAD_DIR)/sqlite-amalgamation-3350000/* pkg/sqlite

clean:
	$(call log, $(YELLOW), "nuking .build/")
	@-rm -rf .build/

# ========================================
# target for common golang tasks

# go build tags used by test, vet and more
TAGS = "libsqlite3,sqlite_vtable,vtable,sqlite_json1,static,system_libgit2"

vet:
	go vet -v -tags=$(TAGS) ./...

build:
	go build -v -tags=$(TAGS) askgit.go

lint:
	golangci-lint run --build-tags $(TAGS)

test:
	go test -v -tags=$(TAGS) ./...

test-cover:
	go test -v -tags=$(TAGS) ./... -cover -covermode=count -coverprofile=coverage.out
	go tool cover -html=coverage.out

bench:
	go test -v -tags=$(TAGS) -bench=. -benchmem -run=^nomatch ./...

# ========================================
# some utility methods

# ASCII color codes that can be used with functions that output to stdout
RED		:= 1;31
GREEN	:= 1;32
ORANGE	:= 1;33
YELLOW	:= 1;33
BLUE	:= 1;34
PURPLE	:= 1;35
CYAN	:= 1;36

# log:
#	print out $2 to stdout using $1 as ASCII color codes
define log
	@printf "\033[$(strip $1)m-- %s\033[0m\n" $2
endef