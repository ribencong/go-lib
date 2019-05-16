BINDIR=bin

#.PHONY: pbs

all: m test
#
#pbs:
#	cd pbs/ && $(MAKE)
#
test:
	 go build -ldflags '-w -s' -o $(BINDIR)/ctest
m:
	CGO_CFLAGS=-mmacosx-version-min=10.11 \
	CGO_LDFLAGS=-mmacosx-version-min=10.11 \
	GOARCH=amd64 GOOS=darwin go build  --buildmode=c-archive -o $(BINDIR)/dss.a mac/*.go

a:
	gomobile bind -v -o $(BINDIR)/dss.aar -target=android github.com/ribencong/go-lib/android

i:
	gomobile bind -v -o $(BINDIR)/dss.framework -target=ios github.com/ribencong/go-lib/ios

clean:
	gomobile clean
	rm $(BINDIR)/*