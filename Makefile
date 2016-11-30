BINARY=cloudo

VERSION=$(shell git describe --tags --always --dirty)
DATE=`date +%FT%T%z`
BUILD=`git rev-parse HEAD`
SOURCE=$(shell find . -type f -name "*.go" | grep -v vendor)
PKGS=$(go list ./... | grep -v /vendor/)
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"

build:$(SOURCE)
	go build ${LDFLAGS} -o ${BINARY} 
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
install:$(SOURCE)
	go install ${LDFLAGS} 
test:$(SOURCE)
	go test -v $(PKGS)

.PHONY: clean install all build 