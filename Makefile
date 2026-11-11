NAME=yaml2srs
BINDIR=build
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo v1.1.0)
GOBUILD=CGO_ENABLED=0 go build --ldflags="-s -w -X github.com/W0n9CNer/yaml2srs/cmd/yaml2srs/flag.version=$(VERSION)"
GOFILE=cmd/yaml2srs/main.go
EXAMPLE_PATH=examples
EXAMPLE_OUTPUT_PATH=${EXAMPLE_PATH}/output

EXAMPLE_FOLDER_PATH=${EXAMPLE_PATH}/folder


EXAMPLE_LINKS_PATH=${EXAMPLE_PATH}/links
EXAMPLE_LINKS_TXT_PATH=${EXAMPLE_LINKS_PATH}/links.txt

.PHONY: clean build linux-amd64 folder links test vet fmt

clean:
	rm -rf ${BINDIR} ${EXAMPLE_OUTPUT_PATH}

build: linux-amd64

linux-amd64:
	GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@ $(GOFILE)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

folder:
	$(BINDIR)/$(NAME)-linux-amd64 folder -p ${EXAMPLE_FOLDER_PATH} -o ${EXAMPLE_OUTPUT_PATH}

links:
	$(BINDIR)/$(NAME)-linux-amd64 links -p ${EXAMPLE_LINKS_TXT_PATH} -o ${EXAMPLE_OUTPUT_PATH}
