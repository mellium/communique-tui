GOFILES != find . -name '*.go'
VERSION!=git describe --tags --dirty
COMMIT!=git rev-parse --short HEAD 2>/dev/null
GO?=go
TAGS?=

LDFLAGS =-X main.Commit=$(COMMIT)
LDFLAGS+=-X main.Version=$(VERSION)

communiqué: go.mod go.sum $(GOFILES)
	$(GO) build \
		-tags "$(TAGS)" \
		-o $@ \
		-ldflags "$(LDFLAGS)"
