VERSION := $(shell git describe 2>/dev/null)

ifndef VERSION
	VERSION = 0.0.1
endif

LDFLAGS := $(LDFLAGS) -s -w -X main.version=${VERSION}

.PHONY: help
help:
	@echo 'Targets:'
	@echo '  all        - download dependencies and compile straw binary'
	@echo '  deps       - download dependencies'
	@echo '  build      - compile straw binary'
	@echo '  test       - run short unit tests'
	@echo '  fmt        - format source files'
	@echo '  tidy       - tidy go modules'
	@echo '  check-deps - check docs/LICENSE_OF_DEPENDENCIES.md'
	@echo '  clean      - delete build artifacts'
	@echo ''
	@echo 'Package Targets:'
	@$(foreach dist,$(dists),echo "  $(dist)";)

.PHONY: all
all:
	@$(MAKE) deps
	@$(MAKE) build

.PHONY: deps
deps:
	go mod download

.PHONY: info
info:
	@echo $(VERSION)
	@echo $(LDFLAGS)

.PHONY: build
build:
	@echo "Build straw"
	GOOS=linux \
	GOARCH=amd64
	go build -o straw -ldflags "$(LDFLAGS)" ./cmd/straw

.PHONY: test
test:
	go test -short $(race_detector) ./...

.PHONY: tidy
tidy:
	go mod verify
	go mod tidy
	@if ! git diff --quiet go.mod go.sum; then \
		echo "please run go mod tidy and check in changes"; \
		exit 1; \
	fi

.PHONY: check-deps
check-deps:
	./scripts/check-deps.sh

.PHONY: clean
clean:
	rm -f straw
	rm -f straw.exe
	rm -rf build