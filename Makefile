PREFIX?=/usr/local
BINDIR?=${PREFIX}/bin
MANDIR?=${PREFIX}/share/man
GOFILES!=find . -name '*.go' ! -name 'catalog.go'
LOCALES!=find ./locales -name '*.json'
VERSION!=git describe --tags --dirty 2>/dev/null | grep . || echo "devel"
COMMIT!=git rev-parse --short HEAD 2>/dev/null
LDFLAGS=-X main.Commit=$(COMMIT) -X main.Version=$(VERSION)
GO=go
TAGS=

communiqué: go.mod go.sum catalog.go $(GOFILES)
	$(GO) build \
		-trimpath \
		-tags "$(TAGS)" \
		-o $@ \
		-ldflags "$(LDFLAGS)"

catalog.go: $(LOCALES) $(GOFILES)
	go generate -run="gotext" .

.PHONY: install
install: communiqué communiqué.1
	install -d ${DESTDIR}${BINDIR} ${DESTDIR}${MANDIR}/man1
	install communiqué ${DESTDIR}${BINDIR}
	install -m 644 communiqué.1 ${DESTDIR}${MANDIR}/man1
