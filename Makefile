VERSION := $(shell git describe 2>/dev/null)

ifndef VERSION
	VERSION = 0.0.1
endif

LDFLAGS := $(LDFLAGS) -s -w -X main.version=${VERSION}

.PHONY: info
info:
	@echo $(VERSION)
	@echo $(LDFLAGS)

.PHONY: build
build:
	@echo "Build straw"
	GOOS=linux \
	GOARCH=amd64
	go build -o straw -ldflags "$(LDFLAGS)" cmd/straw.go
