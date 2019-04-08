BINDIR=bin

.PHONY: pbs

all: pbs lib test

pbs:
	cd pbs/ && $(MAKE)
test:
	 go build -ldflags '-w -s' -o $(BINDIR)/ctest
lib:
	@if [ -z "$(shell which go)" ]; then echo "error: Go must be installed (golang.org)."; exit 1; fi
	CGO_CFLAGS=-mmacosx-version-min=10.11 \
	CGO_LDFLAGS=-mmacosx-version-min=10.11 \
	GOARCH=amd64 GOOS=darwin go build --buildmode=c-archive -o $(BINDIR)/dss.a

clean:
	rm $(BINDIR)/*